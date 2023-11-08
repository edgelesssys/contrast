package kbs

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"go.step.sm/crypto/jose"
)

type server struct {
	http.Handler

	cookieToTEEPubKey map[string]*jose.JSONWebKey
	cookieToNonce     map[string]string
	privKey           *rsa.PrivateKey
	certGen           meshCertGenerator
}

type meshCertGenerator interface {
	NewMeshCert() ([]byte, []byte, error)
}

func NewHandler(ca meshCertGenerator) (*server, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	pemdata := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PUBLIC KEY",
			Bytes: x509.MarshalPKCS1PublicKey(&privateKey.PublicKey),
		},
	)
	log.Println("server's public key:")
	fmt.Println(string(pemdata))

	server := &server{
		cookieToTEEPubKey: make(map[string]*jose.JSONWebKey),
		cookieToNonce:     make(map[string]string),
		privKey:           privateKey,
		certGen:           ca,
	}
	server.Handler = server.newHandler()

	return server, nil
}

func (s *server) newHandler() *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/kbs/v0/resource/{repository}/{type}/{tag}", s.GetResourceHandler).Methods(http.MethodGet)
	r.HandleFunc("/kbs/v0/auth", s.AuthHandler).Methods(http.MethodPost)
	r.HandleFunc("/kbs/v0/attest", s.AttestHandler).Methods(http.MethodPost)
	// We don't implement the following calls of the KBS API.
	r.HandleFunc("/kbs/v0/attestation-policy", s.NotImplementedHandler)
	r.HandleFunc("/kbs/v0/token-certificate-chain", s.NotImplementedHandler)
	return r
}

func (s *server) GetResourceHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("GetResourceHandler called")
	sessionIDCookie, err := r.Cookie(sessionIDCookieName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	log.Printf("%s: %q\n", sessionIDCookieName, sessionIDCookie.Value)

	teePubKey, ok := s.cookieToTEEPubKey[sessionIDCookie.Value]
	if !ok {
		http.Error(w, "invalid session cookie", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	repository := vars["repository"]
	resourceType := vars["type"]
	tag := vars["tag"]
	log.Printf("request path: repository: %q, type: %q, tag: %q\n", repository, resourceType, tag)

	recipient := jose.Recipient{Algorithm: jose.RSA_OAEP, Key: teePubKey}
	encrypter, err := jose.NewEncrypter(jose.A128GCM, recipient, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	cert, key, err := s.certGen.NewMeshCert()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	meshID := struct {
		cert []byte
		key  []byte
	}{
		cert: cert,
		key:  key,
	}

	meshIDJSON, err := json.Marshal(meshID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	object, err := encrypter.Encrypt(meshIDJSON)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	serialized := object.FullSerialize()

	if _, err := w.Write([]byte(serialized)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

const sessionIDCookieName = "kbs-session-id"

func (s *server) AuthHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("AuthHandler called")

	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	log.Printf("req: %+v\n", req)

	// TODO: validate and use request

	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	challenge := Challenge{Nonce: base64.StdEncoding.EncodeToString(nonce)}
	cookieUUID := uuid.New().String()

	cookie := http.Cookie{
		Name:  sessionIDCookieName,
		Value: cookieUUID,
	}

	http.SetCookie(w, &cookie)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(challenge); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.cookieToNonce[cookieUUID] = challenge.Nonce
}

func (s *server) AttestHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("AttestHandler called")
	sessionIDCookie, err := r.Cookie(sessionIDCookieName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	log.Printf("%s: %q\n", sessionIDCookieName, sessionIDCookie.Value)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	log.Printf("body: %s\n", string(body))

	var req Attestation
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if !req.TEEPubKey.Valid() {
		http.Error(w, "invalid TEE pub key", http.StatusBadRequest)
		return
	}
	s.cookieToTEEPubKey[sessionIDCookie.Value] = &req.TEEPubKey

	token := jwt.NewWithClaims(
		jwt.SigningMethodRS256,
		jwt.MapClaims{
			"exp":               time.Now().Add(time.Hour * 72).Unix(),
			"iat":               time.Now().Unix(),
			"iss":               "coordinator-kbs",
			"tee-pubkey":        "pub",
			"tcb-status":        "claims",
			"evaluation-report": "report",
		},
	)

	// Sign and get the complete encoded token as a string using the secret
	tokenString, err := token.SignedString(s.privKey)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("answering with token:")
	fmt.Println(tokenString)

	if _, err := w.Write([]byte(tokenString)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *server) NotImplementedHandler(w http.ResponseWriter, r *http.Request) {
	err := fmt.Errorf("%s not implemented", r.URL.Path)
	log.Println(err)
	http.Error(w, err.Error(), http.StatusNotImplemented)
}

type Request struct {
	Version     string `json:"version"`
	TEE         string `json:"tee"`
	ExtraParams string `json:"extra-params"`
}

type Challenge struct {
	Nonce       string `json:"nonce"`
	ExtraParams string `json:"extra-params"`
}

type Attestation struct {
	TEEPubKey   jose.JSONWebKey `json:"tee-pubkey"`
	TEEEvidence string          `json:"tee-evidence"`
}

type Response struct { // jwt...
	Protected    string `json:"protected"`
	EncryptedKey string `json:"encrypted_key"`
	IV           string `json:"iv"`
	Ciphertext   string `json:"ciphertext"`
	Tag          string `json:"tag"`
}
