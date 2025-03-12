"use strict";(self.webpackChunkcontrast_docs=self.webpackChunkcontrast_docs||[]).push([[2132],{84505:(e,n,t)=>{t.r(n),t.d(n,{assets:()=>c,contentTitle:()=>a,default:()=>h,frontMatter:()=>o,metadata:()=>i,toc:()=>d});const i=JSON.parse('{"id":"architecture/security-considerations","title":"Security Considerations","description":"Contrast ensures application integrity and provides secure means of communication and bootstrapping (see security benefits).","source":"@site/docs/architecture/security-considerations.md","sourceDirName":"architecture","slug":"/architecture/security-considerations","permalink":"/contrast/pr-preview/pr-1292/next/architecture/security-considerations","draft":false,"unlisted":false,"editUrl":"https://github.com/edgelesssys/contrast/edit/main/docs/docs/architecture/security-considerations.md","tags":[],"version":"current","frontMatter":{},"sidebar":"docs","previous":{"title":"Certificate authority","permalink":"/contrast/pr-preview/pr-1292/next/architecture/certificates"},"next":{"title":"Observability","permalink":"/contrast/pr-preview/pr-1292/next/architecture/observability"}}');var s=t(74848),r=t(28453);const o={},a="Security Considerations",c={},d=[{value:"General recommendations",id:"general-recommendations",level:2},{value:"Authentication",id:"authentication",level:3},{value:"Encryption",id:"encryption",level:3},{value:"Contrast security guarantees",id:"contrast-security-guarantees",level:2},{value:"Limitations inherent to policy checking",id:"limitations-inherent-to-policy-checking",level:3},{value:"Logs",id:"logs",level:3}];function l(e){const n={a:"a",code:"code",h1:"h1",h2:"h2",h3:"h3",header:"header",li:"li",ol:"ol",p:"p",pre:"pre",...(0,r.R)(),...e.components};return(0,s.jsxs)(s.Fragment,{children:[(0,s.jsx)(n.header,{children:(0,s.jsx)(n.h1,{id:"security-considerations",children:"Security Considerations"})}),"\n",(0,s.jsxs)(n.p,{children:["Contrast ensures application integrity and provides secure means of communication and bootstrapping (see ",(0,s.jsx)(n.a,{href:"/contrast/pr-preview/pr-1292/next/basics/security-benefits",children:"security benefits"}),").\nHowever, care must be taken when interacting with the outside of Contrast's confidential environment.\nThis page presents some tips for writing secure applications and outlines the trust boundaries app developers need to know."]}),"\n",(0,s.jsx)(n.h2,{id:"general-recommendations",children:"General recommendations"}),"\n",(0,s.jsx)(n.h3,{id:"authentication",children:"Authentication"}),"\n",(0,s.jsx)(n.p,{children:"The application receives credentials from the Contrast Coordinator during initialization.\nThis allows to authenticate towards peers and to verify credentials received from peers.\nThe application should use the certificate bundle to authenticate incoming requests and be wary of unauthenticated requests or requests with a different root of trust (for example the internet PKI)."}),"\n",(0,s.jsx)(n.p,{children:"The recommendation to authenticate not only applies to network traffic, but also to volumes, GPUs and other devices.\nGenerally speaking, all information provided by the world outside the confidential VM should be treated with due scepticism, especially if it's not authenticated.\nCommon cases where Kubernetes apps interact with external services include DNS, Kubernetes API clients and cloud storage endpoints."}),"\n",(0,s.jsx)(n.h3,{id:"encryption",children:"Encryption"}),"\n",(0,s.jsx)(n.p,{children:"Any external persistence should be encrypted with an authenticated cipher.\nThis recommendation applies to block devices or filesystems mounted into the container, but also to cloud blob storage or external databases."}),"\n",(0,s.jsx)(n.h2,{id:"contrast-security-guarantees",children:"Contrast security guarantees"}),"\n",(0,s.jsx)(n.p,{children:"If an application authenticates with a certificate signed by the Contrast Mesh CA of a given manifest, Contrast provides the following guarantees:"}),"\n",(0,s.jsxs)(n.ol,{children:["\n",(0,s.jsx)(n.li,{children:"The container images used by the app are the images specified in the resource definitions."}),"\n",(0,s.jsx)(n.li,{children:"The command line arguments of containers are exactly the arguments specified in the resource definitions."}),"\n",(0,s.jsx)(n.li,{children:"All environment variables are either specified in resource definitions, in the container image manifest or in a settings file for the Contrast CLI."}),"\n",(0,s.jsx)(n.li,{children:"The containers run in a confidential VM that matches the reference values in the manifest."}),"\n",(0,s.jsx)(n.li,{children:"The containers' root filesystems are mounted in encrypted memory."}),"\n"]}),"\n",(0,s.jsx)(n.h3,{id:"limitations-inherent-to-policy-checking",children:"Limitations inherent to policy checking"}),"\n",(0,s.jsx)(n.p,{children:"Workload policies serve as workload identities.\nFrom the perspective of the Contrast Coordinator, all workloads that authenticate with the same policy are equal.\nThus, it's not possible to disambiguate, for example, pods spawned from a deployment or to limit the amount of certificates issued per policy."}),"\n",(0,s.jsxs)(n.p,{children:["Container image references from Kubernetes resource definitions are taken into account when generating the policy.\nA mutable reference may lead to policy failures or unverified image content, depending on the Contrast runtime.\nReliability and security can only be ensured with a full image reference, including digest.\nThe ",(0,s.jsxs)(n.a,{href:"https://docs.docker.com/reference/cli/docker/image/pull/#pull-an-image-by-digest-immutable-identifier",children:[(0,s.jsx)(n.code,{children:"docker pull"})," documentation"]})," explains pinned image references in detail."]}),"\n",(0,s.jsxs)(n.p,{children:["Policies can only verify what can be inferred at generation time.\nSome attributes of Kubernetes pods can't be predicted and thus can't be verified.\nParticularly the ",(0,s.jsx)(n.a,{href:"https://kubernetes.io/docs/concepts/workloads/pods/downward-api/",children:"downward API"})," contains many fields that are dynamic or depend on the host environment, rendering it unsafe for process environment or arguments.\nThe same goes for ",(0,s.jsx)(n.code,{children:"ConfigMap"})," and ",(0,s.jsx)(n.code,{children:"Secret"})," resources, which can also be used to populate container fields.\nIf the application requires such external information, it should be injected as a mount point and carefully inspected before use."]}),"\n",(0,s.jsx)(n.p,{children:"Another type of dynamic content are persistent volumes.\nAny volumes mounted to the pod need to be scrutinized, and sensitive data must not be written to unprotected volumes.\nIdeally, a volume is mounted as a raw block device and authenticated encryption is added within the confidential container."}),"\n",(0,s.jsx)(n.h3,{id:"logs",children:"Logs"}),"\n",(0,s.jsxs)(n.p,{children:["By default, container logs are visible to the host to enable normal Kubernetes operations, for example debugging using ",(0,s.jsx)(n.code,{children:"kubectl logs"}),".\nThe application needs to ensure that sensitive information isn't logged."]}),"\n",(0,s.jsxs)(n.p,{children:["If logs access isn't required, it can be denied with a manual change to the policy settings.\nAfter the initial run of ",(0,s.jsx)(n.code,{children:"contrast generate"}),", there will be a ",(0,s.jsx)(n.code,{children:"settings.json"})," file in the working directory.\nModify the default for ",(0,s.jsx)(n.code,{children:"ReadStreamRequest"})," like shown in the diff below and run ",(0,s.jsx)(n.code,{children:"contrast generate"})," again."]}),"\n",(0,s.jsx)(n.pre,{children:(0,s.jsx)(n.code,{className:"language-diff",children:'diff --git a/settings.json b/settings-no-logs.json\nindex fd998a4..6760000 100644\n--- a/settings.json\n+++ b/settings-no-logs.json\n@@ -330,7 +330,7 @@\n             "regex": []\n         },\n         "CloseStdinRequest": false,\n-        "ReadStreamRequest": true,\n+        "ReadStreamRequest": false,\n         "UpdateEphemeralMountsRequest": false,\n         "WriteStreamRequest": false\n     }\n\n'})})]})}function h(e={}){const{wrapper:n}={...(0,r.R)(),...e.components};return n?(0,s.jsx)(n,{...e,children:(0,s.jsx)(l,{...e})}):l(e)}},28453:(e,n,t)=>{t.d(n,{R:()=>o,x:()=>a});var i=t(96540);const s={},r=i.createContext(s);function o(e){const n=i.useContext(r);return i.useMemo((function(){return"function"==typeof e?e(n):{...n,...e}}),[n,e])}function a(e){let n;return n=e.disableParentContext?"function"==typeof e.components?e.components(s):e.components||s:o(e.components),i.createElement(r.Provider,{value:n},e.children)}}}]);