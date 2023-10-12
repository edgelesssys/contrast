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

	teePubKeys    map[string]*jose.JSONWebKey
	cookieToNonce map[string]string
	privKey       *rsa.PrivateKey
}

func NewHandler() (*server, error) {
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
		teePubKeys:    make(map[string]*jose.JSONWebKey),
		cookieToNonce: make(map[string]string),
		privKey:       privateKey,
	}
	server.Handler = server.newHandler()

	return server, nil
}

func (s *server) newHandler() *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/kbs/v0/resource/{repository}/{type}/{tag}", s.GetResourceHandler).Methods(http.MethodGet)
	r.HandleFunc("/kbs/v0/auth", s.AuthHandler).Methods(http.MethodPost)
	r.HandleFunc("/kbs/v0/attest", s.AttestHandler)
	r.HandleFunc("/kbs/v0/attestation-policy", s.AttestationPolicyHandler)
	r.HandleFunc("/kbs/v0/token-certificate-chain", s.TokenCertificateCainHandler)
	return r
}

func (s *server) GetResourceHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("GetResourceHandler called")
	cookie := r.Header.Get("Cookie")
	log.Printf("cookie: %q\n", cookie)

	vars := mux.Vars(r)
	repository := vars["repository"]
	resourceType := vars["type"]
	tag := vars["tag"]
	log.Printf("repository: %q, type: %q, tag: %q\n", repository, resourceType, tag)

	teePubKey, ok := s.teePubKeys[cookie]
	if !ok {
		panic("no tee key found")
	}

	recipient := jose.Recipient{Algorithm: jose.RSA_OAEP, Key: teePubKey}
	encrypter, err := jose.NewEncrypter(jose.A128GCM, recipient, nil)
	if err != nil {
		panic(err)
	}
	plaintext := []byte("Lorem ipsum dolor sit amet")
	object, err := encrypter.Encrypt(plaintext)
	if err != nil {
		panic(err)
	}
	serialized := object.FullSerialize()

	if _, err := w.Write([]byte(serialized)); err != nil {
		panic(err)
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
	s.teePubKeys[sessionIDCookie.Value] = &req.TEEPubKey

	token := jwt.NewWithClaims(
		jwt.SigningMethodRS256,
		jwt.MapClaims{
			"exp":        time.Now().Add(time.Hour * 72).Unix(),
			"iat":        time.Now().Unix(),
			"iss":        "coco-fake-kbs",
			"tee-pubkey": "foo",
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

func (s *server) AttestationPolicyHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("PolicyHandler called")
	log.Printf("cookie: %q\n", r.Header.Get("Cookie"))
	body, err := io.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}
	log.Printf("body: %s\n", string(body))

	w.Header().Set("Content-Type", "application/json")
}

func (s *server) TokenCertificateCainHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("TokenCertificateCainHandler called")
	log.Printf("cookie: %q\n", r.Header.Get("Cookie"))

	w.Header().Set("Content-Type", "application/json")
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
