"use strict";(self.webpackChunkcontrast_docs=self.webpackChunkcontrast_docs||[]).push([[8772],{74153:(e,t,r)=>{r.r(t),r.d(t,{assets:()=>d,contentTitle:()=>a,default:()=>l,frontMatter:()=>o,metadata:()=>s,toc:()=>c});const s=JSON.parse('{"id":"architecture/secrets","title":"Secrets & recovery","description":"When the Coordinator is configured with the initial manifest, it generates a random secret seed.","source":"@site/versioned_docs/version-1.1/architecture/secrets.md","sourceDirName":"architecture","slug":"/architecture/secrets","permalink":"/contrast/pr-preview/pr-1120/1.1/architecture/secrets","draft":false,"unlisted":false,"editUrl":"https://github.com/edgelesssys/contrast/edit/main/docs/versioned_docs/version-1.1/architecture/secrets.md","tags":[],"version":"1.1","frontMatter":{},"sidebar":"docs","previous":{"title":"Attestation","permalink":"/contrast/pr-preview/pr-1120/1.1/architecture/attestation"},"next":{"title":"Certificate authority","permalink":"/contrast/pr-preview/pr-1120/1.1/architecture/certificates"}}');var n=r(74848),i=r(28453);const o={},a="Secrets & recovery",d={},c=[{value:"Persistence",id:"persistence",level:2},{value:"Recovery",id:"recovery",level:2},{value:"Workload Secrets",id:"workload-secrets",level:2}];function h(e){const t={admonition:"admonition",code:"code",h1:"h1",h2:"h2",header:"header",p:"p",...(0,i.R)(),...e.components};return(0,n.jsxs)(n.Fragment,{children:[(0,n.jsx)(t.header,{children:(0,n.jsx)(t.h1,{id:"secrets--recovery",children:"Secrets & recovery"})}),"\n",(0,n.jsx)(t.p,{children:"When the Coordinator is configured with the initial manifest, it generates a random secret seed.\nFrom this seed, it uses an HKDF to derive the CA root key and a signing key for the manifest history.\nThis derivation is deterministic, so the seed can be used to bring any Coordinator to this Coordinator's state."}),"\n",(0,n.jsxs)(t.p,{children:["The secret seed is returned to the user on the first call to ",(0,n.jsx)(t.code,{children:"contrast set"}),", encrypted with the user's public seed share owner key.\nIf no seed share owner key is provided, a key is generated and stored in the working directory."]}),"\n",(0,n.jsx)(t.h2,{id:"persistence",children:"Persistence"}),"\n",(0,n.jsxs)(t.p,{children:["The Coordinator runs as a ",(0,n.jsx)(t.code,{children:"StatefulSet"})," with a dynamically provisioned persistent volume.\nThis volume stores the manifest history and the associated runtime policies.\nThe manifest isn't considered sensitive information, because it needs to be passed to the untrusted infrastructure in order to start workloads.\nHowever, the Coordinator must ensure its integrity and that the persisted data corresponds to the manifests set by authorized users.\nThus, the manifest is stored in plain text, but is signed with a private key derived from the Coordinator's secret seed."]}),"\n",(0,n.jsx)(t.h2,{id:"recovery",children:"Recovery"}),"\n",(0,n.jsxs)(t.p,{children:["When a Coordinator starts up, it doesn't have access to the signing secret and can thus not verify the integrity of the persisted manifests.\nIt needs to be provided with the secret seed, from which it can derive the signing key that verifies the signatures.\nThis procedure is called recovery and is initiated by the workload owner.\nThe CLI decrypts the secret seed using the private seed share owner key, verifies the Coordinator and sends the seed through the ",(0,n.jsx)(t.code,{children:"Recover"})," method.\nThe Coordinator recovers its key material and verifies the manifest history signature."]}),"\n",(0,n.jsx)(t.h2,{id:"workload-secrets",children:"Workload Secrets"}),"\n",(0,n.jsxs)(t.p,{children:["The Coordinator provides each workload a secret seed during attestation. This secret can be used by the workload to derive additional secrets for example to\nencrypt persistent data. Like the workload certificates it's mounted in the shared Kubernetes volume ",(0,n.jsx)(t.code,{children:"contrast-secrets"})," in the path ",(0,n.jsx)(t.code,{children:"<mountpoint>/secrets/workload-secret-seed"}),"."]}),"\n",(0,n.jsx)(t.admonition,{type:"warning",children:(0,n.jsx)(t.p,{children:"The workload owner can decrypt data encrypted with secrets derived from the workload secret.\nThe workload owner can derive the workload secret themselves, since it's derived from the secret seed known to the workload owner.\nIf the data owner and the workload owner is the same entity, then they can safely use the workload secrets."})})]})}function l(e={}){const{wrapper:t}={...(0,i.R)(),...e.components};return t?(0,n.jsx)(t,{...e,children:(0,n.jsx)(h,{...e})}):h(e)}},28453:(e,t,r)=>{r.d(t,{R:()=>o,x:()=>a});var s=r(96540);const n={},i=s.createContext(n);function o(e){const t=s.useContext(i);return s.useMemo((function(){return"function"==typeof e?e(t):{...t,...e}}),[t,e])}function a(e){let t;return t=e.disableParentContext?"function"==typeof e.components?e.components(n):e.components||n:o(e.components),s.createElement(i.Provider,{value:t},e.children)}}}]);