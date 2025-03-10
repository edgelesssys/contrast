"use strict";(self.webpackChunkcontrast_docs=self.webpackChunkcontrast_docs||[]).push([[2550],{90721:(e,t,i)=>{i.r(t),i.d(t,{assets:()=>c,contentTitle:()=>o,default:()=>f,frontMatter:()=>s,metadata:()=>r,toc:()=>h});const r=JSON.parse('{"id":"architecture/certificates","title":"Certificate authority","description":"The Coordinator acts as a certificate authority (CA) for the workloads","source":"@site/docs/architecture/certificates.md","sourceDirName":"architecture","slug":"/architecture/certificates","permalink":"/contrast/pr-preview/pr-1280/next/architecture/certificates","draft":false,"unlisted":false,"editUrl":"https://github.com/edgelesssys/contrast/edit/main/docs/docs/architecture/certificates.md","tags":[],"version":"current","frontMatter":{},"sidebar":"docs","previous":{"title":"Secrets & recovery","permalink":"/contrast/pr-preview/pr-1280/next/architecture/secrets"},"next":{"title":"Security considerations","permalink":"/contrast/pr-preview/pr-1280/next/architecture/security-considerations"}}');var n=i(74848),a=i(28453);const s={},o="Certificate authority",c={},h=[{value:"Public key infrastructure",id:"public-key-infrastructure",level:2},{value:"Certificate rotation",id:"certificate-rotation",level:2},{value:"Usage of the different certificates",id:"usage-of-the-different-certificates",level:2}];function d(e){const t={em:"em",h1:"h1",h2:"h2",header:"header",img:"img",li:"li",p:"p",strong:"strong",ul:"ul",...(0,a.R)(),...e.components};return(0,n.jsxs)(n.Fragment,{children:[(0,n.jsx)(t.header,{children:(0,n.jsx)(t.h1,{id:"certificate-authority",children:"Certificate authority"})}),"\n",(0,n.jsx)(t.p,{children:"The Coordinator acts as a certificate authority (CA) for the workloads\ndefined in the manifest.\nAfter a workload pod's attestation has been verified by the Coordinator,\nit receives a mesh certificate and the mesh CA certificate.\nThe mesh certificate can be used for example in a TLS connection as the server or\nclient certificate to proof to the other party that the workload has been\nverified by the Coordinator. The other party can verify the mesh certificate\nwith the mesh CA certificate. While the certificates can be used by the workload\ndeveloper in different ways, they're automatically used in Contrast's service\nmesh to establish mTLS connections between workloads in the same deployment."}),"\n",(0,n.jsx)(t.h2,{id:"public-key-infrastructure",children:"Public key infrastructure"}),"\n",(0,n.jsx)(t.p,{children:"The Coordinator establishes a public key infrastructure (PKI) for all workloads\ncontained in the manifest. The Coordinator holds three certificates: the root CA\ncertificate, the intermediate CA certificate, and the mesh CA certificate.\nThe root CA certificate is a long-lasting certificate and its private key signs\nthe intermediate CA certificate. The intermediate CA certificate and the mesh CA\ncertificate share the same private key. This intermediate private key is used\nto sign the mesh certificates. Moreover, the intermediate private key and\ntherefore the intermediate CA certificate and the mesh CA certificate are\nrotated when setting a new manifest."}),"\n",(0,n.jsx)(t.p,{children:(0,n.jsx)(t.img,{alt:"PKI certificate chain",src:i(34102).A+""})}),"\n",(0,n.jsx)(t.h2,{id:"certificate-rotation",children:"Certificate rotation"}),"\n",(0,n.jsx)(t.p,{children:"Depending on the configuration of the first manifest, it allows the workload\nowner to update the manifest and, therefore, the deployment.\nWorkload owners and data owners can be mutually untrusted parties.\nTo protect against the workload owner silently introducing malicious containers,\nthe Coordinator rotates the intermediate private key every time the manifest is\nupdated and, therefore, the\nintermediate CA certificate and mesh CA certificate. If the user doesn't\ntrust the workload owner, they use the mesh CA certificate obtained when they\nverified the Coordinator and the manifest. This ensures that the user only\nconnects to workloads defined in the manifest they verified since only those\nworkloads' certificates are signed with this intermediate private key."}),"\n",(0,n.jsx)(t.p,{children:"Similarly, the service mesh also uses the mesh CA certificate obtained when the\nworkload was started, so the workload only trusts endpoints that have been\nverified by the Coordinator based on the same manifest. Consequently, a\nmanifest update requires a fresh rollout of the services in the service mesh."}),"\n",(0,n.jsx)(t.h2,{id:"usage-of-the-different-certificates",children:"Usage of the different certificates"}),"\n",(0,n.jsxs)(t.ul,{children:["\n",(0,n.jsxs)(t.li,{children:["The ",(0,n.jsx)(t.strong,{children:"root CA certificate"})," is returned when verifying the Coordinator.\nThe data owner can use it to verify the mesh certificates of the workloads.\nThis should only be used if the data owner trusts all future updates to the\nmanifest and workloads. This is, for instance, the case when the workload owner is\nthe same entity as the data owner."]}),"\n",(0,n.jsxs)(t.li,{children:["The ",(0,n.jsx)(t.strong,{children:"mesh CA certificate"})," is returned when verifying the Coordinator.\nThe data owner can use it to verify the mesh certificates of the workloads.\nThis certificate is bound to the manifest set when the Coordinator is verified.\nIf the manifest is updated, the mesh CA certificate changes.\nNew workloads will receive mesh certificates signed by the ",(0,n.jsx)(t.em,{children:"new"})," mesh CA  certificate.\nThe Coordinator with the new manifest needs to be verified to retrieve the new mesh CA certificate.\nThe service mesh also uses the mesh CA certificate to verify the mesh certificates."]}),"\n",(0,n.jsxs)(t.li,{children:["The ",(0,n.jsx)(t.strong,{children:"intermediate CA certificate"})," links the root CA certificate to the\nmesh certificate so that the mesh certificate can be verified with the root CA\ncertificate. It's part of the certificate chain handed out by\nendpoints in the service mesh."]}),"\n",(0,n.jsxs)(t.li,{children:["The ",(0,n.jsx)(t.strong,{children:"mesh certificate"})," is part of the certificate chain handed out by\nendpoints in the service mesh. During the startup of a pod, the Initializer\nrequests a certificate from the Coordinator. This mesh certificate will be returned if the Coordinator successfully\nverifies the workload. The mesh certificate\ncontains X.509 extensions with information from the workloads attestation\ndocument."]}),"\n"]})]})}function f(e={}){const{wrapper:t}={...(0,a.R)(),...e.components};return t?(0,n.jsx)(t,{...e,children:(0,n.jsx)(d,{...e})}):d(e)}},34102:(e,t,i)=>{i.d(t,{A:()=>r});const r=i.p+"assets/images/contrast_pki.drawio-a2442a1eeb081612c5ad587a58589ad4.svg"},28453:(e,t,i)=>{i.d(t,{R:()=>s,x:()=>o});var r=i(96540);const n={},a=r.createContext(n);function s(e){const t=r.useContext(a);return r.useMemo((function(){return"function"==typeof e?e(t):{...t,...e}}),[t,e])}function o(e){let t;return t=e.disableParentContext?"function"==typeof e.components?e.components(n):e.components||n:s(e.components),r.createElement(a.Provider,{value:t},e.children)}}}]);