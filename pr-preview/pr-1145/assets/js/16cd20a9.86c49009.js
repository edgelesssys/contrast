"use strict";(self.webpackChunkcontrast_docs=self.webpackChunkcontrast_docs||[]).push([[2007],{7254:(e,t,n)=>{n.r(t),n.d(t,{assets:()=>c,contentTitle:()=>a,default:()=>h,frontMatter:()=>o,metadata:()=>i,toc:()=>l});const i=JSON.parse('{"id":"components/policies","title":"Policies","description":"Kata runtime policies are an integral part of the Confidential Containers preview on AKS.","source":"@site/versioned_docs/version-1.2/components/policies.md","sourceDirName":"components","slug":"/components/policies","permalink":"/contrast/pr-preview/pr-1145/1.2/components/policies","draft":false,"unlisted":false,"editUrl":"https://github.com/edgelesssys/contrast/edit/main/docs/versioned_docs/version-1.2/components/policies.md","tags":[],"version":"1.2","frontMatter":{},"sidebar":"docs","previous":{"title":"Runtime","permalink":"/contrast/pr-preview/pr-1145/1.2/components/runtime"},"next":{"title":"Service mesh","permalink":"/contrast/pr-preview/pr-1145/1.2/components/service-mesh"}}');var s=n(74848),r=n(28453);const o={},a="Policies",c={},l=[{value:"Structure",id:"structure",level:2},{value:"Generation",id:"generation",level:2},{value:"Evaluation",id:"evaluation",level:2},{value:"Guarantees",id:"guarantees",level:2},{value:"Trust",id:"trust",level:2},{value:"Platform Differences",id:"platform-differences",level:2}];function d(e){const t={a:"a",code:"code",em:"em",h1:"h1",h2:"h2",header:"header",li:"li",ol:"ol",p:"p",ul:"ul",...(0,r.R)(),...e.components};return(0,s.jsxs)(s.Fragment,{children:[(0,s.jsx)(t.header,{children:(0,s.jsx)(t.h1,{id:"policies",children:"Policies"})}),"\n",(0,s.jsx)(t.p,{children:"Kata runtime policies are an integral part of the Confidential Containers preview on AKS.\nThey prescribe how a Kubernetes pod must be configured to launch successfully in a confidential VM.\nIn Contrast, policies act as a workload identifier: only pods with a policy registered in the manifest receive workload certificates and may participate in the confidential deployment.\nVerification of the Contrast Coordinator and its manifest transitively guarantees that all workloads meet the owner's expectations."}),"\n",(0,s.jsx)(t.h2,{id:"structure",children:"Structure"}),"\n",(0,s.jsxs)(t.p,{children:["The Kata agent running in the confidential micro-VM exposes an RPC service ",(0,s.jsx)(t.a,{href:"https://github.com/kata-containers/kata-containers/blob/e5e0983/src/libs/protocols/protos/agent.proto#L21-L76",children:(0,s.jsx)(t.code,{children:"AgentService"})})," to the Kata runtime.\nThis service handles potentially untrustworthy requests from outside the TCB, which need to be checked against a policy."]}),"\n",(0,s.jsxs)(t.p,{children:["Kata runtime policies are written in the policy language ",(0,s.jsx)(t.a,{href:"https://www.openpolicyagent.org/docs/latest/policy-language/",children:"Rego"}),".\nThey specify what ",(0,s.jsx)(t.code,{children:"AgentService"})," methods can be called, and the permissible parameters for each call."]}),"\n",(0,s.jsxs)(t.p,{children:["Policies consist of two parts: a list of rules and a data section.\nWhile the list of rules is static, the data section is populated with information from the ",(0,s.jsx)(t.code,{children:"PodSpec"})," and other sources."]}),"\n",(0,s.jsx)(t.h2,{id:"generation",children:"Generation"}),"\n",(0,s.jsxs)(t.p,{children:["Runtime policies are programmatically generated from Kubernetes manifests by the Contrast CLI.\nThe ",(0,s.jsx)(t.code,{children:"generate"})," subcommand inspects pod definitions and derives rules for validating the pod at the Kata agent.\nThere are two important integrity checks: container image checksums and OCI runtime parameters."]}),"\n",(0,s.jsxs)(t.p,{children:["For each of the container images used in a pod, the CLI downloads all image layers and produces a cryptographic ",(0,s.jsx)(t.a,{href:"https://www.kernel.org/doc/html/latest/admin-guide/device-mapper/verity.html",children:"dm-verity"})," checksum.\nThese checksums are the basis for the policy's ",(0,s.jsx)(t.em,{children:"storage data"}),"."]}),"\n",(0,s.jsxs)(t.p,{children:["The CLI combines information from the ",(0,s.jsx)(t.code,{children:"PodSpec"}),", ",(0,s.jsx)(t.code,{children:"ConfigMaps"}),", and ",(0,s.jsx)(t.code,{children:"Secrets"})," in the provided Kubernetes manifests to derive a permissible set of command-line arguments and environment variables.\nThese constitute the policy's ",(0,s.jsx)(t.em,{children:"OCI data"}),"."]}),"\n",(0,s.jsx)(t.h2,{id:"evaluation",children:"Evaluation"}),"\n",(0,s.jsxs)(t.p,{children:["The generated policy document is annotated to the pod definitions in Base64 encoding.\nThis annotation is propagated to the Kata runtime, which calculates the SHA256 checksum for the policy and uses that as SNP ",(0,s.jsx)(t.code,{children:"HOSTDATA"})," or TDX ",(0,s.jsx)(t.code,{children:"MRCONFIGID"})," for the confidential micro-VM."]}),"\n",(0,s.jsxs)(t.p,{children:["After the VM launched, the runtime calls the agent's ",(0,s.jsx)(t.code,{children:"SetPolicy"})," method with the full policy document.\nIf the policy doesn't match the checksum in ",(0,s.jsx)(t.code,{children:"HOSTDATA"})," or ",(0,s.jsx)(t.code,{children:"MRCONFIGID"}),", the agent rejects the policy.\nOtherwise, it applies the policy to all future ",(0,s.jsx)(t.code,{children:"AgentService"})," requests."]}),"\n",(0,s.jsx)(t.h2,{id:"guarantees",children:"Guarantees"}),"\n",(0,s.jsx)(t.p,{children:"The policy evaluation provides the following guarantees for pods launched with the correct generated policy:"}),"\n",(0,s.jsxs)(t.ul,{children:["\n",(0,s.jsx)(t.li,{children:"Command and its arguments are set as specified in the resources."}),"\n",(0,s.jsx)(t.li,{children:"There are no unexpected additional environment variables."}),"\n",(0,s.jsx)(t.li,{children:"The container image layers correspond to the layers observed at policy generation time.\nThus, only the expected workload image can be instantiated."}),"\n",(0,s.jsx)(t.li,{children:"Executing additional processes in a container is prohibited."}),"\n",(0,s.jsx)(t.li,{children:"Sending data to a container's standard input is prohibited."}),"\n"]}),"\n",(0,s.jsx)(t.p,{children:"The current implementation of policy checking has some blind spots:"}),"\n",(0,s.jsxs)(t.ul,{children:["\n",(0,s.jsx)(t.li,{children:"Containers can be started in any order, or be omitted entirely."}),"\n",(0,s.jsx)(t.li,{children:"Environment variables may be missing."}),"\n",(0,s.jsxs)(t.li,{children:["Volumes other than the container root volume don't have integrity checks (particularly relevant for mounted ",(0,s.jsx)(t.code,{children:"ConfigMaps"})," and ",(0,s.jsx)(t.code,{children:"Secrets"}),")."]}),"\n"]}),"\n",(0,s.jsx)(t.h2,{id:"trust",children:"Trust"}),"\n",(0,s.jsx)(t.p,{children:"Contrast verifies its confidential containers following these steps:"}),"\n",(0,s.jsxs)(t.ol,{children:["\n",(0,s.jsx)(t.li,{children:"The Contrast CLI generates a policy and attaches it to the pod definition."}),"\n",(0,s.jsx)(t.li,{children:"Kubernetes schedules the pod on a node with the confidential computing runtime."}),"\n",(0,s.jsx)(t.li,{children:"Containerd invokes the Kata runtime to create the pod sandbox."}),"\n",(0,s.jsxs)(t.li,{children:["The Kata runtime starts a CVM with the policy's digest as ",(0,s.jsx)(t.code,{children:"HOSTDATA"}),"/",(0,s.jsx)(t.code,{children:"MRCONFIGID"}),"."]}),"\n",(0,s.jsxs)(t.li,{children:["The Kata runtime sets the policy using the ",(0,s.jsx)(t.code,{children:"SetPolicy"})," method."]}),"\n",(0,s.jsxs)(t.li,{children:["The Kata agent verifies that the incoming policy's digest matches ",(0,s.jsx)(t.code,{children:"HOSTDATA"}),"/",(0,s.jsx)(t.code,{children:"MRCONFIGID"}),"."]}),"\n",(0,s.jsx)(t.li,{children:"The CLI sets a manifest in the Contrast Coordinator, including a list of permitted policies."}),"\n",(0,s.jsx)(t.li,{children:"The Contrast Initializer sends an attestation report to the Contrast Coordinator, asking for a mesh certificate."}),"\n",(0,s.jsxs)(t.li,{children:["The Contrast Coordinator verifies that the started pod has a permitted policy hash in its ",(0,s.jsx)(t.code,{children:"HOSTDATA"}),"/",(0,s.jsx)(t.code,{children:"MRCONFIGID"})," field."]}),"\n"]}),"\n",(0,s.jsx)(t.p,{children:"After the last step, we know that the policy hasn't been tampered with and, thus, that the workload matches expectations and may receive mesh certificates."}),"\n",(0,s.jsx)(t.h2,{id:"platform-differences",children:"Platform Differences"}),"\n",(0,s.jsx)(t.p,{children:"Contrast uses different rules and data sections for different platforms.\nThis results in different policy hashes for different platforms."}),"\n",(0,s.jsxs)(t.p,{children:["The ",(0,s.jsx)(t.code,{children:"generate"})," command automatically derives the correct set of rules and data sections from the ",(0,s.jsx)(t.code,{children:"reference-values"})," flag."]}),"\n",(0,s.jsxs)(t.p,{children:["The ",(0,s.jsx)(t.code,{children:"verify"}),", ",(0,s.jsx)(t.code,{children:"set"}),", and ",(0,s.jsx)(t.code,{children:"recover"})," commands need to know the coordinator's expected policy hash to verify its identity.\nBy default these commands assume that the coordinator is using the policy for the ",(0,s.jsx)(t.code,{children:"AKS-CLH-SNP"})," platform.\nIf the coordinator is running on a different platform, the correct policy hash can be looked up in the ",(0,s.jsx)(t.code,{children:"coordinator-policy.hash"})," file bundled with the ",(0,s.jsx)(t.a,{href:"https://github.com/edgelesssys/contrast/releases",children:"Contrast release"}),".\nThe coordinator policy hash can be overwritten using the ",(0,s.jsx)(t.code,{children:"--coordinator-policy-hash"})," flag."]})]})}function h(e={}){const{wrapper:t}={...(0,r.R)(),...e.components};return t?(0,s.jsx)(t,{...e,children:(0,s.jsx)(d,{...e})}):d(e)}},28453:(e,t,n)=>{n.d(t,{R:()=>o,x:()=>a});var i=n(96540);const s={},r=i.createContext(s);function o(e){const t=i.useContext(r);return i.useMemo((function(){return"function"==typeof e?e(t):{...t,...e}}),[t,e])}function a(e){let t;return t=e.disableParentContext?"function"==typeof e.components?e.components(s):e.components||s:o(e.components),i.createElement(r.Provider,{value:t},e.children)}}}]);