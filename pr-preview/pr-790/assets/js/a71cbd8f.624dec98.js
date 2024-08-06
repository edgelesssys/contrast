"use strict";(self.webpackChunkcontrast_docs=self.webpackChunkcontrast_docs||[]).push([[5003],{4973:(e,t,r)=>{r.r(t),r.d(t,{assets:()=>a,contentTitle:()=>o,default:()=>u,frontMatter:()=>n,metadata:()=>c,toc:()=>d});var s=r(4848),i=r(8453);const n={},o="Secrets & recovery",c={id:"architecture/secrets",title:"Secrets & recovery",description:"When the Coordinator is configured with the initial manifest, it generates a random secret seed.",source:"@site/versioned_docs/version-0.8/architecture/secrets.md",sourceDirName:"architecture",slug:"/architecture/secrets",permalink:"/contrast/pr-preview/pr-790/architecture/secrets",draft:!1,unlisted:!1,editUrl:"https://github.com/edgelesssys/contrast/edit/main/docs/versioned_docs/version-0.8/architecture/secrets.md",tags:[],version:"0.8",frontMatter:{},sidebar:"docs",previous:{title:"Attestation",permalink:"/contrast/pr-preview/pr-790/architecture/attestation"},next:{title:"Certificate authority",permalink:"/contrast/pr-preview/pr-790/architecture/certificates"}},a={},d=[{value:"Persistence",id:"persistence",level:2},{value:"Recovery",id:"recovery",level:2}];function h(e){const t={code:"code",h1:"h1",h2:"h2",p:"p",...(0,i.R)(),...e.components};return(0,s.jsxs)(s.Fragment,{children:[(0,s.jsx)(t.h1,{id:"secrets--recovery",children:"Secrets & recovery"}),"\n",(0,s.jsx)(t.p,{children:"When the Coordinator is configured with the initial manifest, it generates a random secret seed.\nFrom this seed, it uses an HKDF to derive the CA root key and a signing key for the manifest history.\nThis derivation is deterministic, so the seed can be used to bring any Coordinator to this Coordinator's state."}),"\n",(0,s.jsxs)(t.p,{children:["The secret seed is returned to the user on the first call to ",(0,s.jsx)(t.code,{children:"contrast set"}),", encrypted with the user's public seed share owner key.\nIf no seed share owner key is provided, a key is generated and stored in the working directory."]}),"\n",(0,s.jsx)(t.h2,{id:"persistence",children:"Persistence"}),"\n",(0,s.jsxs)(t.p,{children:["The Coordinator runs as a ",(0,s.jsx)(t.code,{children:"StatefulSet"})," with a dynamically provisioned persistent volume.\nThis volume stores the manifest history and the associated runtime policies.\nThe manifest isn't considered sensitive information, because it needs to be passed to the untrusted infrastructure in order to start workloads.\nHowever, the Coordinator must ensure its integrity and that the persisted data corresponds to the manifests set by authorized users.\nThus, the manifest is stored in plain text, but is signed with a private key derived from the Coordinator's secret seed."]}),"\n",(0,s.jsx)(t.h2,{id:"recovery",children:"Recovery"}),"\n",(0,s.jsxs)(t.p,{children:["When a Coordinator starts up, it doesn't have access to the signing secret and can thus not verify the integrity of the persisted manifests.\nIt needs to be provided with the secret seed, from which it can derive the signing key that verifies the signatures.\nThis procedure is called recovery and is initiated by the workload owner.\nThe CLI decrypts the secret seed using the private seed share owner key, verifies the Coordinator and sends the seed through the ",(0,s.jsx)(t.code,{children:"Recover"})," method.\nThe Coordinator recovers its key material and verifies the manifest history signature."]})]})}function u(e={}){const{wrapper:t}={...(0,i.R)(),...e.components};return t?(0,s.jsx)(t,{...e,children:(0,s.jsx)(h,{...e})}):h(e)}},8453:(e,t,r)=>{r.d(t,{R:()=>o,x:()=>c});var s=r(6540);const i={},n=s.createContext(i);function o(e){const t=s.useContext(n);return s.useMemo((function(){return"function"==typeof e?e(t):{...t,...e}}),[t,e])}function c(e){let t;return t=e.disableParentContext?"function"==typeof e.components?e.components(i):e.components||i:o(e.components),s.createElement(n.Provider,{value:t},e.children)}}}]);