"use strict";(self.webpackChunkcontrast_docs=self.webpackChunkcontrast_docs||[]).push([[426],{67156:(e,t,n)=>{n.r(t),n.d(t,{assets:()=>c,contentTitle:()=>s,default:()=>h,frontMatter:()=>r,metadata:()=>a,toc:()=>d});var i=n(74848),o=n(28453);const r={},s="Components",a={id:"components/overview",title:"Components",description:"Contrast is composed of several key components that work together to manage and scale confidential containers effectively within Kubernetes environments.",source:"@site/versioned_docs/version-0.8/components/overview.md",sourceDirName:"components",slug:"/components/overview",permalink:"/contrast/pr-preview/pr-932/0.8/components/overview",draft:!1,unlisted:!1,editUrl:"https://github.com/edgelesssys/contrast/edit/main/docs/versioned_docs/version-0.8/components/overview.md",tags:[],version:"0.8",frontMatter:{},sidebar:"docs",previous:{title:"Troubleshooting",permalink:"/contrast/pr-preview/pr-932/0.8/troubleshooting"},next:{title:"Runtime",permalink:"/contrast/pr-preview/pr-932/0.8/components/runtime"}},c={},d=[{value:"The CLI (Command Line Interface)",id:"the-cli-command-line-interface",level:2},{value:"The Coordinator",id:"the-coordinator",level:2},{value:"The Manifest",id:"the-manifest",level:2},{value:"Runtime policies",id:"runtime-policies",level:2},{value:"The Initializer",id:"the-initializer",level:2},{value:"The Contrast runtime",id:"the-contrast-runtime",level:2}];function l(e){const t={a:"a",code:"code",em:"em",h1:"h1",h2:"h2",header:"header",img:"img",li:"li",p:"p",ul:"ul",...(0,o.R)(),...e.components};return(0,i.jsxs)(i.Fragment,{children:[(0,i.jsx)(t.header,{children:(0,i.jsx)(t.h1,{id:"components",children:"Components"})}),"\n",(0,i.jsx)(t.p,{children:"Contrast is composed of several key components that work together to manage and scale confidential containers effectively within Kubernetes environments.\nThis page provides an overview of the core components essential for deploying and managing Contrast."}),"\n",(0,i.jsx)(t.p,{children:(0,i.jsx)(t.img,{alt:"components overview",src:n(91175).A+"",width:"3387",height:"1866"})}),"\n",(0,i.jsx)(t.h2,{id:"the-cli-command-line-interface",children:"The CLI (Command Line Interface)"}),"\n",(0,i.jsx)(t.p,{children:"The CLI serves as the primary management tool for Contrast deployments. It's designed to streamline the configuration and operation of Contrast in several ways:"}),"\n",(0,i.jsxs)(t.ul,{children:["\n",(0,i.jsx)(t.li,{children:"Installation and setup: The CLI facilitates the installation of the necessary runtime classes required for Contrast to function within a Kubernetes cluster."}),"\n",(0,i.jsx)(t.li,{children:"Policy generation: It allows users to generate runtime policies, adapt the deployment files, and generate the Contrast manifest."}),"\n",(0,i.jsx)(t.li,{children:"Configuration management: Through the CLI, users can configure the Contrast Coordinator with the generated manifest."}),"\n",(0,i.jsx)(t.li,{children:"Verification and attestation: The CLI provides tools to verify the integrity and authenticity of the Coordinator and the entire deployment via remote attestation."}),"\n"]}),"\n",(0,i.jsx)(t.h2,{id:"the-coordinator",children:"The Coordinator"}),"\n",(0,i.jsxs)(t.p,{children:["The Contrast Coordinator is the central remote attestation service of a Contrast deployment.\nIt runs inside a confidential container inside your cluster.\nThe Coordinator can be verified via remote attestation, and a Contrast deployment is self-contained.\nThe Coordinator is configured with a ",(0,i.jsx)(t.em,{children:"manifest"}),", a configuration file containing the reference attestation values of your deployment.\nIt ensures that your deployment's topology adheres to your specified manifest by verifying the identity and integrity of all confidential pods inside the deployment.\nThe Coordinator is also a certificate authority and issues certificates for your workload pods during the attestation procedure.\nYour workload pods can establish secure, encrypted communication channels between themselves based on these certificates using the Coordinator as the root CA.\nAs your app needs to scale, the Coordinator transparently verifies new instances and then provides them with their certificates to join the deployment."]}),"\n",(0,i.jsx)(t.p,{children:"To verify your deployment, the Coordinator's remote attestation statement combined with the manifest offers a concise single remote attestation statement for your entire deployment.\nA third party can use this to verify the integrity of your distributed app, making it easy to assure stakeholders of your app's identity and integrity."}),"\n",(0,i.jsx)(t.h2,{id:"the-manifest",children:"The Manifest"}),"\n",(0,i.jsx)(t.p,{children:"The manifest is the configuration file for the Coordinator, defining your confidential deployment.\nIt's automatically generated from your deployment by the Contrast CLI.\nIt currently consists of the following parts:"}),"\n",(0,i.jsxs)(t.ul,{children:["\n",(0,i.jsxs)(t.li,{children:[(0,i.jsx)(t.em,{children:"Policies"}),": The identities of your Pods, represented by the hashes of their respective runtime policies."]}),"\n",(0,i.jsxs)(t.li,{children:[(0,i.jsx)(t.em,{children:"Reference Values"}),": The remote attestation reference values for the Kata confidential micro-VM that's the runtime environment of your Pods."]}),"\n",(0,i.jsxs)(t.li,{children:[(0,i.jsx)(t.em,{children:"WorkloadOwnerKeyDigest"}),": The workload owner's public key digest. Used for authenticating subsequent manifest updates."]}),"\n"]}),"\n",(0,i.jsx)(t.h2,{id:"runtime-policies",children:"Runtime policies"}),"\n",(0,i.jsx)(t.p,{children:"Runtime Policies are a mechanism to enable the use of the untrusted Kubernetes API for orchestration while ensuring the confidentiality and integrity of your confidential containers.\nThey allow us to enforce the integrity of your containers' runtime environment as defined in your deployment files.\nThe runtime policy mechanism is based on the Open Policy Agent (OPA) and translates the Kubernetes deployment YAML into the Rego policy language of OPA.\nThe Kata Agent inside the confidential micro-VM then enforces the policy by only acting on permitted requests.\nThe Contrast CLI provides the tooling for automatically translating Kubernetes deployment YAML into the Rego policy language of OPA."}),"\n",(0,i.jsx)(t.h2,{id:"the-initializer",children:"The Initializer"}),"\n",(0,i.jsxs)(t.p,{children:["Contrast provides an Initializer that handles the remote attestation on the workload side transparently and\nfetches the workload certificate. The Initializer runs as an init container before your workload is started.\nIt provides the workload container and the ",(0,i.jsx)(t.a,{href:"/contrast/pr-preview/pr-932/0.8/components/service-mesh",children:"service mesh sidecar"})," with the workload certificates."]}),"\n",(0,i.jsx)(t.h2,{id:"the-contrast-runtime",children:"The Contrast runtime"}),"\n",(0,i.jsxs)(t.p,{children:["Contrast depends on a Kubernetes ",(0,i.jsx)(t.a,{href:"https://kubernetes.io/docs/concepts/containers/runtime-class/",children:"runtime class"}),", which is installed\nby the ",(0,i.jsx)(t.code,{children:"node-installer"})," DaemonSet.\nThis runtime consists of a containerd runtime plugin, a virtual machine manager (cloud-hypervisor), and a podvm image (IGVM and rootFS).\nThe installer takes care of provisioning every node in the cluster so it provides this runtime class."]})]})}function h(e={}){const{wrapper:t}={...(0,o.R)(),...e.components};return t?(0,i.jsx)(t,{...e,children:(0,i.jsx)(l,{...e})}):l(e)}},91175:(e,t,n)=>{n.d(t,{A:()=>i});const i=n.p+"assets/images/components-ea3fb9800d5718d6ab96607ee817c104.svg"},28453:(e,t,n)=>{n.d(t,{R:()=>s,x:()=>a});var i=n(96540);const o={},r=i.createContext(o);function s(e){const t=i.useContext(r);return i.useMemo((function(){return"function"==typeof e?e(t):{...t,...e}}),[t,e])}function a(e){let t;return t=e.disableParentContext?"function"==typeof e.components?e.components(o):e.components||o:s(e.components),i.createElement(r.Provider,{value:t},e.children)}}}]);