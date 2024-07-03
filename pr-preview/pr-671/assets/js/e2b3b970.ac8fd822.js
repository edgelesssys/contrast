"use strict";(self.webpackChunkcontrast_docs=self.webpackChunkcontrast_docs||[]).push([[2841],{4251:(e,n,t)=>{t.r(n),t.d(n,{assets:()=>l,contentTitle:()=>o,default:()=>u,frontMatter:()=>r,metadata:()=>a,toc:()=>c});var i=t(4848),s=t(8453);const r={},o="Planned features and limitations",a={id:"features-limitations",title:"Planned features and limitations",description:"This section lists planned features and current limitations of Contrast.",source:"@site/versioned_docs/version-0.7/features-limitations.md",sourceDirName:".",slug:"/features-limitations",permalink:"/contrast/pr-preview/pr-671/features-limitations",draft:!1,unlisted:!1,editUrl:"https://github.com/edgelesssys/contrast/edit/main/docs/versioned_docs/version-0.7/features-limitations.md",tags:[],version:"0.7",frontMatter:{},sidebar:"docs",previous:{title:"Observability",permalink:"/contrast/pr-preview/pr-671/architecture/observability"},next:{title:"About",permalink:"/contrast/pr-preview/pr-671/about/"}},l={},c=[{value:"Availability",id:"availability",level:2},{value:"Kubernetes Features",id:"kubernetes-features",level:2},{value:"Runtime Policies",id:"runtime-policies",level:2},{value:"Tooling Integration",id:"tooling-integration",level:2}];function d(e){const n={a:"a",admonition:"admonition",code:"code",em:"em",h1:"h1",h2:"h2",li:"li",p:"p",strong:"strong",ul:"ul",...(0,s.R)(),...e.components};return(0,i.jsxs)(i.Fragment,{children:[(0,i.jsx)(n.h1,{id:"planned-features-and-limitations",children:"Planned features and limitations"}),"\n",(0,i.jsx)(n.p,{children:"This section lists planned features and current limitations of Contrast."}),"\n",(0,i.jsx)(n.h2,{id:"availability",children:"Availability"}),"\n",(0,i.jsxs)(n.ul,{children:["\n",(0,i.jsxs)(n.li,{children:[(0,i.jsx)(n.strong,{children:"Platform Support"}),": At present, Contrast is exclusively available on Azure AKS, supported by the ",(0,i.jsx)(n.a,{href:"https://learn.microsoft.com/en-us/azure/confidential-computing/confidential-containers-on-aks-preview",children:"Confidential Container preview for AKS"}),". Expansion to other cloud platforms is planned, pending the availability of necessary infrastructure enhancements."]}),"\n",(0,i.jsxs)(n.li,{children:[(0,i.jsx)(n.strong,{children:"Bare Metal Support"}),": Support for running Contrast on Bare Metal Kubernetes will be available soon for AMD SEV and Intel TDX."]}),"\n"]}),"\n",(0,i.jsx)(n.h2,{id:"kubernetes-features",children:"Kubernetes Features"}),"\n",(0,i.jsxs)(n.ul,{children:["\n",(0,i.jsxs)(n.li,{children:[(0,i.jsx)(n.strong,{children:"Persistent Volumes"}),": Not currently supported within Confidential Containers."]}),"\n",(0,i.jsxs)(n.li,{children:[(0,i.jsx)(n.strong,{children:"Port-Forwarding"}),": This feature ",(0,i.jsx)(n.a,{href:"https://github.com/kata-containers/kata-containers/issues/1693",children:"isn't yet supported by Kata Containers"}),". You can ",(0,i.jsx)(n.a,{href:"https://docs.edgeless.systems/contrast/deployment#connect-to-the-contrast-coordinator",children:"deploy a port-forwarder"})," as a workaround."]}),"\n",(0,i.jsxs)(n.li,{children:[(0,i.jsx)(n.strong,{children:"Resource Limits"}),": There is an existing bug on AKS where container memory limits are incorrectly applied. The current workaround involves using only memory requests instead of limits."]}),"\n"]}),"\n",(0,i.jsx)(n.h2,{id:"runtime-policies",children:"Runtime Policies"}),"\n",(0,i.jsxs)(n.ul,{children:["\n",(0,i.jsxs)(n.li,{children:[(0,i.jsx)(n.strong,{children:"Coverage"}),": While the enforcement of workload policies generally functions well, ",(0,i.jsx)(n.a,{href:"https://github.com/microsoft/kata-containers/releases/tag/3.2.0.azl0.genpolicy",children:"there are scenarios not yet fully covered"}),". It's crucial to review deployments specifically for these edge cases."]}),"\n",(0,i.jsxs)(n.li,{children:[(0,i.jsx)(n.strong,{children:"Order of events"}),": The current policy evaluation mechanism on API requests isn't stateful, so it can't ensure a prescribed order of events. Consequently, there's no guaranteed enforcement that the ",(0,i.jsx)(n.a,{href:"/contrast/pr-preview/pr-671/components/service-mesh",children:"service mesh sidecar"})," container runs ",(0,i.jsx)(n.em,{children:"before"})," the workload container. This order ensures that all traffic between pods is securely encapsulated within TLS connections."]}),"\n",(0,i.jsxs)(n.li,{children:[(0,i.jsx)(n.strong,{children:"Absence of events"}),": Policies can't ensure certain events have happened. A container, such as the ",(0,i.jsx)(n.a,{href:"/contrast/pr-preview/pr-671/components/service-mesh",children:"service mesh sidecar"}),", can be omitted entirely. Environment variables may be missing."]}),"\n",(0,i.jsxs)(n.li,{children:[(0,i.jsx)(n.strong,{children:"Volume integrity checks"}),": While persistent volumes aren't supported yet, integrity checks don't currently cover other objects such as ",(0,i.jsx)(n.code,{children:"ConfigMaps"})," and ",(0,i.jsx)(n.code,{children:"Secrets"}),"."]}),"\n"]}),"\n",(0,i.jsx)(n.admonition,{type:"warning",children:(0,i.jsx)(n.p,{children:"The policy limitations, in particular the missing guarantee that our service mesh sidecar has been started before the workload container affects the service mesh implementation of Contrast. Currently, this requires inspecting the iptables rules on startup or terminating TLS connections in the workload directly."})}),"\n",(0,i.jsx)(n.h2,{id:"tooling-integration",children:"Tooling Integration"}),"\n",(0,i.jsxs)(n.ul,{children:["\n",(0,i.jsxs)(n.li,{children:[(0,i.jsx)(n.strong,{children:"CLI Availability"}),": The CLI tool is currently only available for Linux. This limitation arises because certain upstream dependencies haven't yet been ported to other platforms."]}),"\n"]})]})}function u(e={}){const{wrapper:n}={...(0,s.R)(),...e.components};return n?(0,i.jsx)(n,{...e,children:(0,i.jsx)(d,{...e})}):d(e)}},8453:(e,n,t)=>{t.d(n,{R:()=>o,x:()=>a});var i=t(6540);const s={},r=i.createContext(s);function o(e){const n=i.useContext(r);return i.useMemo((function(){return"function"==typeof e?e(n):{...n,...e}}),[n,e])}function a(e){let n;return n=e.disableParentContext?"function"==typeof e.components?e.components(s):e.components||s:o(e.components),i.createElement(r.Provider,{value:n},e.children)}}}]);