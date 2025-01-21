"use strict";(self.webpackChunkcontrast_docs=self.webpackChunkcontrast_docs||[]).push([[7924],{45725:(e,t,n)=>{n.r(t),n.d(t,{assets:()=>c,contentTitle:()=>o,default:()=>l,frontMatter:()=>a,metadata:()=>i,toc:()=>h});const i=JSON.parse('{"id":"architecture/attestation","title":"Attestation in Contrast","description":"This document describes the attestation architecture of Contrast, adhering to the definitions of Remote ATtestation procedureS (RATS) in RFC 9334.","source":"@site/versioned_docs/version-0.8/architecture/attestation.md","sourceDirName":"architecture","slug":"/architecture/attestation","permalink":"/contrast/pr-preview/pr-1163/0.8/architecture/attestation","draft":false,"unlisted":false,"editUrl":"https://github.com/edgelesssys/contrast/edit/main/docs/versioned_docs/version-0.8/architecture/attestation.md","tags":[],"version":"0.8","frontMatter":{},"sidebar":"docs","previous":{"title":"Service mesh","permalink":"/contrast/pr-preview/pr-1163/0.8/components/service-mesh"},"next":{"title":"Secrets & recovery","permalink":"/contrast/pr-preview/pr-1163/0.8/architecture/secrets"}}');var r=n(74848),s=n(28453);const a={},o="Attestation in Contrast",c={},h=[{value:"Attestation architecture",id:"attestation-architecture",level:2},{value:"Components of Contrast&#39;s attestation",id:"components-of-contrasts-attestation",level:2},{value:"Attester: Application Pods",id:"attester-application-pods",level:3},{value:"Verifier: Coordinator and CLI",id:"verifier-coordinator-and-cli",level:3},{value:"Relying Party: Data owner",id:"relying-party-data-owner",level:3},{value:"Evidence generation and appraisal",id:"evidence-generation-and-appraisal",level:2},{value:"Evidence types and formats",id:"evidence-types-and-formats",level:3},{value:"Appraisal policies for evidence",id:"appraisal-policies-for-evidence",level:3},{value:"Frequently asked questions about attestation in Contrast",id:"frequently-asked-questions-about-attestation-in-contrast",level:2},{value:"What&#39;s the purpose of remote attestation in Contrast?",id:"whats-the-purpose-of-remote-attestation-in-contrast",level:3},{value:"How does Contrast ensure the security of the attestation process?",id:"how-does-contrast-ensure-the-security-of-the-attestation-process",level:3},{value:"What security benefits does attestation provide?",id:"what-security-benefits-does-attestation-provide",level:3},{value:"How can you verify the authenticity of attestation results?",id:"how-can-you-verify-the-authenticity-of-attestation-results",level:3},{value:"How are attestation results used by relying parties?",id:"how-are-attestation-results-used-by-relying-parties",level:3},{value:"Summary",id:"summary",level:2}];function d(e){const t={a:"a",code:"code",em:"em",h1:"h1",h2:"h2",h3:"h3",header:"header",img:"img",li:"li",p:"p",strong:"strong",ul:"ul",...(0,s.R)(),...e.components};return(0,r.jsxs)(r.Fragment,{children:[(0,r.jsx)(t.header,{children:(0,r.jsx)(t.h1,{id:"attestation-in-contrast",children:"Attestation in Contrast"})}),"\n",(0,r.jsxs)(t.p,{children:["This document describes the attestation architecture of Contrast, adhering to the definitions of Remote ATtestation procedureS (RATS) in ",(0,r.jsx)(t.a,{href:"https://www.rfc-editor.org/rfc/rfc9334.html",children:"RFC 9334"}),".\nThe following gives a detailed description of Contrast's attestation architecture.\nAt the end of this document, we included an ",(0,r.jsx)(t.a,{href:"#frequently-asked-questions-about-attestation-in-contrast",children:"FAQ"})," that answers the most common questions regarding attestation in hindsight of the ",(0,r.jsx)(t.a,{href:"/contrast/pr-preview/pr-1163/0.8/basics/security-benefits",children:"security benefits"}),"."]}),"\n",(0,r.jsx)(t.h2,{id:"attestation-architecture",children:"Attestation architecture"}),"\n",(0,r.jsxs)(t.p,{children:["Contrast integrates with the RATS architecture, leveraging their definition of roles and processes including ",(0,r.jsx)(t.em,{children:"Attesters"}),", ",(0,r.jsx)(t.em,{children:"Verifiers"}),", and ",(0,r.jsx)(t.em,{children:"Relying Parties"}),"."]}),"\n",(0,r.jsx)(t.p,{children:(0,r.jsx)(t.img,{alt:"Conceptual attestation architecture",src:n(55732).A+"",width:"592",height:"377"})}),"\n",(0,r.jsxs)(t.p,{children:["Figure 1: Conceptual attestation architecture. Taken from ",(0,r.jsx)(t.a,{href:"https://www.rfc-editor.org/rfc/rfc9334.html#figure-1",children:"RFC 9334"}),"."]}),"\n",(0,r.jsxs)(t.ul,{children:["\n",(0,r.jsxs)(t.li,{children:[(0,r.jsx)(t.strong,{children:"Attester"}),": Assigned to entities that are responsible for creating ",(0,r.jsx)(t.em,{children:"Evidence"})," which is then sent to a ",(0,r.jsx)(t.em,{children:"Verifier"}),"."]}),"\n",(0,r.jsxs)(t.li,{children:[(0,r.jsx)(t.strong,{children:"Verifier"}),": These entities utilize the ",(0,r.jsx)(t.em,{children:"Evidence"}),", ",(0,r.jsx)(t.em,{children:"Reference Values"}),", and ",(0,r.jsx)(t.em,{children:"Endorsements"}),". They assess the trustworthiness of the ",(0,r.jsx)(t.em,{children:"Attester"})," by applying an ",(0,r.jsx)(t.em,{children:"Appraisal Policy"})," for ",(0,r.jsx)(t.em,{children:"Evidence"}),". Following this assessment, ",(0,r.jsx)(t.em,{children:"Verifiers"})," generate ",(0,r.jsx)(t.em,{children:"Attestation Results"})," for use by ",(0,r.jsx)(t.em,{children:"Relying Parties"}),". The ",(0,r.jsx)(t.em,{children:"Appraisal Policy"})," for ",(0,r.jsx)(t.em,{children:"Evidence"})," may be provided by the ",(0,r.jsx)(t.em,{children:"Verifier Owner"}),", programmed into the ",(0,r.jsx)(t.em,{children:"Verifier"}),", or acquired through other means."]}),"\n",(0,r.jsxs)(t.li,{children:[(0,r.jsx)(t.strong,{children:"Relying Party"}),": Assigned to entities that utilize ",(0,r.jsx)(t.em,{children:"Attestation Results"}),', applying their own appraisal policies to make specific decisions, such as authorization decisions. This process is referred to as the "appraisal of Attestation Results." The ',(0,r.jsx)(t.em,{children:"Appraisal Policy"})," for ",(0,r.jsx)(t.em,{children:"Attestation Results"})," might be sourced from the ",(0,r.jsx)(t.em,{children:"Relying Party Owner"}),", configured by the owner, embedded in the ",(0,r.jsx)(t.em,{children:"Relying Party"}),", or obtained through other protocols or mechanisms."]}),"\n"]}),"\n",(0,r.jsx)(t.h2,{id:"components-of-contrasts-attestation",children:"Components of Contrast's attestation"}),"\n",(0,r.jsx)(t.p,{children:"The key components involved in the attestation process of Contrast are detailed below:"}),"\n",(0,r.jsx)(t.h3,{id:"attester-application-pods",children:"Attester: Application Pods"}),"\n",(0,r.jsxs)(t.p,{children:["This includes all Pods of the Contrast deployment that run inside Confidential Containers and generate cryptographic evidence reflecting their current configuration and state.\nTheir evidence is rooted in the ",(0,r.jsx)(t.a,{href:"/contrast/pr-preview/pr-1163/0.8/basics/confidential-containers",children:"hardware measurements"})," from the CPU and their ",(0,r.jsx)(t.a,{href:"/contrast/pr-preview/pr-1163/0.8/components/runtime",children:"confidential VM environment"}),".\nThe details of this evidence are given below in the section on ",(0,r.jsx)(t.a,{href:"#evidence-generation-and-appraisal",children:"evidence generation and appraisal"}),"."]}),"\n",(0,r.jsx)(t.p,{children:(0,r.jsx)(t.img,{alt:"Attestation flow of a confidential pod",src:n(15601).A+"",width:"608",height:"697"})}),"\n",(0,r.jsxs)(t.p,{children:["Figure 2: Attestation flow of a confidential pod. Based on the layered attester graphic in ",(0,r.jsx)(t.a,{href:"https://www.rfc-editor.org/rfc/rfc9334.html#figure-3",children:"RFC 9334"}),"."]}),"\n",(0,r.jsxs)(t.p,{children:["Pods run in Contrast's ",(0,r.jsx)(t.a,{href:"/contrast/pr-preview/pr-1163/0.8/components/runtime",children:"runtime environment"})," (B), effectively within a confidential VM.\nDuring launch, the CPU (A) measures the initial memory content of the confidential VM that contains Contrast's pod-VM image and generates the corresponding attestation evidence.\nThe image is in ",(0,r.jsx)(t.a,{href:"https://github.com/microsoft/igvm",children:"IGVM format"}),", encapsulating all information required to launch a virtual machine, including the kernel, the initramfs, and kernel cmdline.\nThe kernel cmdline contains the root hash for ",(0,r.jsx)(t.a,{href:"https://www.kernel.org/doc/html/latest/admin-guide/device-mapper/verity.html",children:"dm-verity"})," that ensures the integrity of the root filesystem.\nThe root filesystem contains all  components of the container's runtime environment including the ",(0,r.jsx)(t.a,{href:"/contrast/pr-preview/pr-1163/0.8/basics/confidential-containers#kata-containers",children:"guest agent"})," (C)."]}),"\n",(0,r.jsxs)(t.p,{children:["In the userland, the guest agent takes care of enforcing the ",(0,r.jsx)(t.a,{href:"/contrast/pr-preview/pr-1163/0.8/components/overview#runtime-policies",children:"runtime policy"})," of the pod.\nWhile the policy is passed in during the initialization procedure via the host, the evidence for the runtime policy is part of the CPU measurements.\nDuring the ",(0,r.jsx)(t.a,{href:"/contrast/pr-preview/pr-1163/0.8/deployment#generate-policy-annotations-and-manifest",children:"deployment"})," the policy is annotated to the Kubernetes Pod resources.\nOn AMD SEV-SNP the hash of the policy is then added to the attestation report via the ",(0,r.jsx)(t.code,{children:"HOSTDATA"})," field by the hypervisor.\nWhen provided with the policy from the Kata host, the guest agent verifies that the policy's hash matches the one in the ",(0,r.jsx)(t.code,{children:"HOSTDATA"})," field."]}),"\n",(0,r.jsx)(t.p,{children:"In summary a Pod's evidence is the attestation report of the CPU that provides evidence for runtime environment and the runtime policy."}),"\n",(0,r.jsx)(t.h3,{id:"verifier-coordinator-and-cli",children:"Verifier: Coordinator and CLI"}),"\n",(0,r.jsxs)(t.p,{children:["The ",(0,r.jsx)(t.a,{href:"/contrast/pr-preview/pr-1163/0.8/components/overview#the-coordinator",children:"Coordinator"})," acts as a verifier within the Contrast deployment, configured with a ",(0,r.jsx)(t.a,{href:"/contrast/pr-preview/pr-1163/0.8/components/overview#the-manifest",children:"Manifest"})," that defines the reference values and serves as an appraisal policy for all pods in the deployment.\nIt also pulls endorsements from hardware vendors to verify the hardware claims.\nThe Coordinator operates within the cluster as a confidential container and provides similar evidence as any other Pod when it acts as an attester.\nIn RATS terminology, the Coordinator's dual role is defined as a lead attester in a composite device which spans the entire deployment: Coordinator and the workload pods.\nIt collects evidence from other attesters and conveys it to a verifier, generating evidence about the layout of the whole composite device based on the Manifest as the appraisal policy."]}),"\n",(0,r.jsx)(t.p,{children:(0,r.jsx)(t.img,{alt:"Deployment attestation as a composite device",src:n(92084).A+"",width:"576",height:"393"})}),"\n",(0,r.jsxs)(t.p,{children:["Figure 3: Contrast deployment as a composite device. Based on the composite device in ",(0,r.jsx)(t.a,{href:"https://www.rfc-editor.org/rfc/rfc9334.html#figure-4",children:"RFC 9334"}),"."]}),"\n",(0,r.jsxs)(t.p,{children:["The ",(0,r.jsx)(t.a,{href:"/contrast/pr-preview/pr-1163/0.8/components/overview#the-cli-command-line-interface",children:"CLI"})," serves as the verifier for the Coordinator and the entire Contrast deployment, containing the reference values for the Coordinator and the endorsements from hardware vendors.\nThese reference values are built into the CLI during our release process and can be reproduced offline via reproducible builds."]}),"\n",(0,r.jsx)(t.h3,{id:"relying-party-data-owner",children:"Relying Party: Data owner"}),"\n",(0,r.jsxs)(t.p,{children:["A relying party in the Contrast scenario could be, for example, the ",(0,r.jsx)(t.a,{href:"/contrast/pr-preview/pr-1163/0.8/basics/security-benefits",children:"data owner"})," that interacts with the application.\nThe relying party can use the CLI to obtain the attestation results and Contrast's ",(0,r.jsx)(t.a,{href:"/contrast/pr-preview/pr-1163/0.8/architecture/certificates",children:"CA certificates"})," bound to these results.\nThe CA certificates can then be used by the relying party to authenticate the application, for example through TLS connections."]}),"\n",(0,r.jsx)(t.h2,{id:"evidence-generation-and-appraisal",children:"Evidence generation and appraisal"}),"\n",(0,r.jsx)(t.h3,{id:"evidence-types-and-formats",children:"Evidence types and formats"}),"\n",(0,r.jsx)(t.p,{children:"In Contrast, attestation evidence revolves around a hardware-generated attestation report, which contains several critical pieces of information:"}),"\n",(0,r.jsxs)(t.ul,{children:["\n",(0,r.jsxs)(t.li,{children:[(0,r.jsx)(t.strong,{children:"The hardware attestation report"}),": This report includes details such as the chip identifier, platform information, microcode versions, and comprehensive guest measurements. The entire report is signed by the CPU's private key, ensuring the authenticity and integrity of the data provided."]}),"\n",(0,r.jsxs)(t.li,{children:[(0,r.jsx)(t.strong,{children:"The launch measurements"}),": Included within the hardware attestation report, this is a digest generated by the CPU that represents a hash of all initial guest memory pages. This includes essential components like the kernel, initramfs, and the kernel command line. Notably, it incorporates the root filesystem's dm-verity root hash, verifying the integrity of the root filesystem."]}),"\n",(0,r.jsxs)(t.li,{children:[(0,r.jsx)(t.strong,{children:"The runtime policy hash"}),": Also part of the hardware attestation report, this field contains the hash of the Rego policy which dictates all expected API commands and their values from the host to the Kata guest agent. It encompasses crucial settings such as dm-verity hashes for the container image layers, environment variables, and mount points."]}),"\n"]}),"\n",(0,r.jsx)(t.h3,{id:"appraisal-policies-for-evidence",children:"Appraisal policies for evidence"}),"\n",(0,r.jsx)(t.p,{children:"The appraisal of this evidence in Contrast is governed by two main components:"}),"\n",(0,r.jsxs)(t.ul,{children:["\n",(0,r.jsxs)(t.li,{children:[(0,r.jsx)(t.strong,{children:"The Manifest"}),": A JSON file used by the Coordinator to align with reference values. It sets the expectations for runtime policy hashes for each pod and includes what should be reported in the hardware attestation report for each component of the deployment."]}),"\n",(0,r.jsxs)(t.li,{children:[(0,r.jsx)(t.strong,{children:"The CLI's appraisal policy"}),": This policy encompasses expected values of the Coordinator\u2019s guest measurements and its runtime policy. It's embedded into the CLI during the build process and ensures that any discrepancy between the built-in values and those reported by the hardware attestation can be identified and addressed. The integrity of this policy is safeguardable through reproducible builds, allowing verification against the source code reference."]}),"\n"]}),"\n",(0,r.jsx)(t.h2,{id:"frequently-asked-questions-about-attestation-in-contrast",children:"Frequently asked questions about attestation in Contrast"}),"\n",(0,r.jsx)(t.h3,{id:"whats-the-purpose-of-remote-attestation-in-contrast",children:"What's the purpose of remote attestation in Contrast?"}),"\n",(0,r.jsx)(t.p,{children:"Remote attestation in Contrast ensures that software runs within a secure, isolated confidential computing environment.\nThis process certifies that the memory is encrypted and confirms the integrity and authenticity of the software running within the deployment.\nBy validating the runtime environment and the policies enforced on it, Contrast ensures that the system operates in a trustworthy state and hasn't been tampered with."}),"\n",(0,r.jsx)(t.h3,{id:"how-does-contrast-ensure-the-security-of-the-attestation-process",children:"How does Contrast ensure the security of the attestation process?"}),"\n",(0,r.jsx)(t.p,{children:"Contrast leverages hardware-rooted security features such as AMD SEV-SNP to generate cryptographic evidence of a pod\u2019s current state and configuration.\nThis evidence is checked against pre-defined appraisal policies to guarantee that only verified and authorized pods are part of a Contrast deployment."}),"\n",(0,r.jsx)(t.h3,{id:"what-security-benefits-does-attestation-provide",children:"What security benefits does attestation provide?"}),"\n",(0,r.jsxs)(t.p,{children:["Attestation confirms the integrity of the runtime environment and the identity of the workloads.\nIt plays a critical role in preventing unauthorized changes and detecting potential modifications at runtime.\nThe attestation provides integrity and authenticity guarantees, enabling relying parties\u2014such as workload operators or data owners\u2014to confirm the effective protection against potential threats, including malicious cloud insiders, co-tenants, or compromised workload operators.\nMore details on the specific security benefits can be found ",(0,r.jsx)(t.a,{href:"/contrast/pr-preview/pr-1163/0.8/basics/security-benefits",children:"here"}),"."]}),"\n",(0,r.jsx)(t.h3,{id:"how-can-you-verify-the-authenticity-of-attestation-results",children:"How can you verify the authenticity of attestation results?"}),"\n",(0,r.jsx)(t.p,{children:"Attestation results in Contrast are tied to cryptographic proofs generated and signed by the hardware itself.\nThese proofs are then verified using public keys from trusted hardware vendors, ensuring that the results aren't only accurate but also resistant to tampering.\nFor further authenticity verification, all of Contrast's code is reproducibly built, and the attestation evidence can be verified locally from the source code."}),"\n",(0,r.jsx)(t.h3,{id:"how-are-attestation-results-used-by-relying-parties",children:"How are attestation results used by relying parties?"}),"\n",(0,r.jsxs)(t.p,{children:["Relying parties use attestation results to make informed security decisions, such as allowing access to sensitive data or resources only if the attestation verifies the system's integrity.\nThereafter, the use of Contrast's ",(0,r.jsx)(t.a,{href:"/contrast/pr-preview/pr-1163/0.8/architecture/certificates",children:"CA certificates in TLS connections"})," provides a practical approach to communicate securely with the application."]}),"\n",(0,r.jsx)(t.h2,{id:"summary",children:"Summary"}),"\n",(0,r.jsx)(t.p,{children:"In summary, Contrast's attestation strategy adheres to the RATS guidelines and consists of robust verification mechanisms that ensure each component of the deployment is secure and trustworthy.\nThis comprehensive approach allows Contrast to provide a high level of security assurance to its users."})]})}function l(e={}){const{wrapper:t}={...(0,s.R)(),...e.components};return t?(0,r.jsx)(t,{...e,children:(0,r.jsx)(d,{...e})}):d(e)}},92084:(e,t,n)=>{n.d(t,{A:()=>i});const i=n.p+"assets/images/attestation-composite-device-b91e917a07f0f2bb082989317a9b053f.svg"},15601:(e,t,n)=>{n.d(t,{A:()=>i});const i=n.p+"assets/images/attestation-pod-af4fc34dd97b1fdc20f66ecfa4fdeb1a.svg"},55732:(e,t,n)=>{n.d(t,{A:()=>i});const i=n.p+"assets/images/attestation-rats-architecture-1a05fc2165e5c75a171e7b7832186bc3.svg"},28453:(e,t,n)=>{n.d(t,{R:()=>a,x:()=>o});var i=n(96540);const r={},s=i.createContext(r);function a(e){const t=i.useContext(s);return i.useMemo((function(){return"function"==typeof e?e(t):{...t,...e}}),[t,e])}function o(e){let t;return t=e.disableParentContext?"function"==typeof e.components?e.components(r):e.components||r:a(e.components),i.createElement(s.Provider,{value:t},e.children)}}}]);