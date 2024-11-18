"use strict";(self.webpackChunkcontrast_docs=self.webpackChunkcontrast_docs||[]).push([[5811],{26033:(e,n,t)=>{t.r(n),t.d(n,{assets:()=>c,contentTitle:()=>a,default:()=>h,frontMatter:()=>r,metadata:()=>s,toc:()=>d});const s=JSON.parse('{"id":"basics/confidential-containers","title":"Confidential Containers","description":"Contrast uses some building blocks from Confidential Containers (CoCo), a CNCF Sandbox project that aims to standardize confidential computing at the pod level.","source":"@site/versioned_docs/version-1.1/basics/confidential-containers.md","sourceDirName":"basics","slug":"/basics/confidential-containers","permalink":"/contrast/pr-preview/pr-1011/basics/confidential-containers","draft":false,"unlisted":false,"editUrl":"https://github.com/edgelesssys/contrast/edit/main/docs/versioned_docs/version-1.1/basics/confidential-containers.md","tags":[],"version":"1.1","frontMatter":{},"sidebar":"docs","previous":{"title":"What is Contrast?","permalink":"/contrast/pr-preview/pr-1011/"},"next":{"title":"Security benefits","permalink":"/contrast/pr-preview/pr-1011/basics/security-benefits"}}');var i=t(74848),o=t(28453);const r={},a="Confidential Containers",c={},d=[{value:"Kubernetes RuntimeClass",id:"kubernetes-runtimeclass",level:2},{value:"Kata Containers",id:"kata-containers",level:2},{value:"AKS CoCo preview",id:"aks-coco-preview",level:2}];function l(e){const n={a:"a",code:"code",h1:"h1",h2:"h2",header:"header",p:"p",...(0,o.R)(),...e.components};return(0,i.jsxs)(i.Fragment,{children:[(0,i.jsx)(n.header,{children:(0,i.jsx)(n.h1,{id:"confidential-containers",children:"Confidential Containers"})}),"\n",(0,i.jsxs)(n.p,{children:["Contrast uses some building blocks from ",(0,i.jsx)(n.a,{href:"https://confidentialcontainers.org",children:"Confidential Containers"})," (CoCo), a ",(0,i.jsx)(n.a,{href:"https://www.cncf.io/projects/confidential-containers/",children:"CNCF Sandbox project"})," that aims to standardize confidential computing at the pod level.\nThe project is under active development and many of the high-level features are still in flux.\nContrast uses the more stable core primitive provided by CoCo: its Kubernetes runtime."]}),"\n",(0,i.jsx)(n.h2,{id:"kubernetes-runtimeclass",children:"Kubernetes RuntimeClass"}),"\n",(0,i.jsxs)(n.p,{children:["Kubernetes can be extended to use more than one container runtime with ",(0,i.jsx)(n.a,{href:"https://kubernetes.io/docs/concepts/containers/runtime-class/",children:(0,i.jsx)(n.code,{children:"RuntimeClass"})})," objects.\nThe ",(0,i.jsx)(n.a,{href:"https://kubernetes.io/docs/concepts/architecture/cri/",children:"Container Runtime Interface"})," (CRI) implementation, for example containerd, dispatches pod management API calls to the appropriate ",(0,i.jsx)(n.code,{children:"RuntimeClass"}),".\n",(0,i.jsx)(n.code,{children:"RuntimeClass"})," implementations are usually based on an ",(0,i.jsx)(n.a,{href:"https://github.com/opencontainers/runtime-spec",children:"OCI runtime"}),", such as ",(0,i.jsx)(n.code,{children:"runc"}),", ",(0,i.jsx)(n.code,{children:"runsc"})," or ",(0,i.jsx)(n.code,{children:"crun"}),".\nIn CoCo's case, the runtime is Kata Containers with added confidential computing capabilities."]}),"\n",(0,i.jsx)(n.h2,{id:"kata-containers",children:"Kata Containers"}),"\n",(0,i.jsxs)(n.p,{children:[(0,i.jsx)(n.a,{href:"https://katacontainers.io/",children:"Kata Containers"})," is an OCI runtime that runs pods in VMs.\nThe pod VM spawns an agent process that accepts management commands from the Kata runtime running on the host.\nThere are two options for creating pod VMs: local to the Kubernetes node, or remote VMs created with cloud provider APIs.\nUsing local VMs requires either bare-metal servers or VMs with support for nested virtualization.\nLocal VMs communicate with the host over a virtual socket.\nFor remote VMs, host-to-agent communication is tunnelled through the cloud provider's network."]}),"\n",(0,i.jsxs)(n.p,{children:["Kata Containers was originally designed to isolate the guest from the host, but it can also run pods in confidential VMs (CVMs) to shield pods from their underlying infrastructure.\nIn confidential mode, the guest agent is configured with an ",(0,i.jsx)(n.a,{href:"https://www.openpolicyagent.org/",children:"Open Policy Agent"})," (OPA) policy to authorize API calls from the host.\nThis policy also contains checksums for the expected container images.\nIt's derived from Kubernetes resource definitions and its checksum is included in the attestation report."]}),"\n",(0,i.jsx)(n.h2,{id:"aks-coco-preview",children:"AKS CoCo preview"}),"\n",(0,i.jsxs)(n.p,{children:[(0,i.jsx)(n.a,{href:"https://learn.microsoft.com/en-us/azure/aks/",children:"Azure Kubernetes Service"})," (AKS) provides CoCo-enabled node pools as a ",(0,i.jsx)(n.a,{href:"https://learn.microsoft.com/en-us/azure/aks/confidential-containers-overview",children:"preview offering"}),".\nThese node pools leverage Azure VM types capable of nested virtualization (CVM-in-VM) and the CoCo stack is pre-installed.\nContrast can be deployed directly into a CoCo-enabled AKS cluster."]})]})}function h(e={}){const{wrapper:n}={...(0,o.R)(),...e.components};return n?(0,i.jsx)(n,{...e,children:(0,i.jsx)(l,{...e})}):l(e)}},28453:(e,n,t)=>{t.d(n,{R:()=>r,x:()=>a});var s=t(96540);const i={},o=s.createContext(i);function r(e){const n=s.useContext(o);return s.useMemo((function(){return"function"==typeof e?e(n):{...n,...e}}),[n,e])}function a(e){let n;return n=e.disableParentContext?"function"==typeof e.components?e.components(i):e.components||i:r(e.components),s.createElement(o.Provider,{value:n},e.children)}}}]);