"use strict";(self.webpackChunkcontrast_docs=self.webpackChunkcontrast_docs||[]).push([[8902],{97232:(e,n,t)=>{t.r(n),t.d(n,{assets:()=>l,contentTitle:()=>a,default:()=>h,frontMatter:()=>o,metadata:()=>i,toc:()=>c});const i=JSON.parse('{"id":"features-limitations","title":"Planned features and limitations","description":"This section lists planned features and current limitations of Contrast.","source":"@site/versioned_docs/version-0.8/features-limitations.md","sourceDirName":".","slug":"/features-limitations","permalink":"/contrast/pr-preview/pr-994/0.8/features-limitations","draft":false,"unlisted":false,"editUrl":"https://github.com/edgelesssys/contrast/edit/main/docs/versioned_docs/version-0.8/features-limitations.md","tags":[],"version":"0.8","frontMatter":{},"sidebar":"docs","previous":{"title":"Observability","permalink":"/contrast/pr-preview/pr-994/0.8/architecture/observability"},"next":{"title":"Telemetry","permalink":"/contrast/pr-preview/pr-994/0.8/about/telemetry"}}');var r=t(74848),s=t(28453);const o={},a="Planned features and limitations",l={},c=[{value:"Availability",id:"availability",level:2},{value:"Kubernetes features",id:"kubernetes-features",level:2},{value:"Runtime policies",id:"runtime-policies",level:2},{value:"Tooling integration",id:"tooling-integration",level:2},{value:"Automatic recovery and high availability",id:"automatic-recovery-and-high-availability",level:2}];function d(e){const n={a:"a",admonition:"admonition",code:"code",em:"em",h1:"h1",h2:"h2",header:"header",li:"li",p:"p",strong:"strong",ul:"ul",...(0,s.R)(),...e.components};return(0,r.jsxs)(r.Fragment,{children:[(0,r.jsx)(n.header,{children:(0,r.jsx)(n.h1,{id:"planned-features-and-limitations",children:"Planned features and limitations"})}),"\n",(0,r.jsx)(n.p,{children:"This section lists planned features and current limitations of Contrast."}),"\n",(0,r.jsx)(n.h2,{id:"availability",children:"Availability"}),"\n",(0,r.jsxs)(n.ul,{children:["\n",(0,r.jsxs)(n.li,{children:[(0,r.jsx)(n.strong,{children:"Platform support"}),": At present, Contrast is exclusively available on Azure AKS, supported by the ",(0,r.jsx)(n.a,{href:"https://learn.microsoft.com/en-us/azure/confidential-computing/confidential-containers-on-aks-preview",children:"Confidential Container preview for AKS"}),". Expansion to other cloud platforms is planned, pending the availability of necessary infrastructure enhancements."]}),"\n",(0,r.jsxs)(n.li,{children:[(0,r.jsx)(n.strong,{children:"Bare-metal support"}),": Support for running Contrast on bare-metal Kubernetes will be available soon for AMD SEV and Intel TDX."]}),"\n"]}),"\n",(0,r.jsx)(n.h2,{id:"kubernetes-features",children:"Kubernetes features"}),"\n",(0,r.jsxs)(n.ul,{children:["\n",(0,r.jsxs)(n.li,{children:[(0,r.jsx)(n.strong,{children:"Persistent volumes"}),": Contrast only supports volumes with ",(0,r.jsx)(n.a,{href:"https://kubernetes.io/docs/concepts/storage/persistent-volumes/#volume-mode",children:(0,r.jsx)(n.code,{children:"volumeMode: Block"})}),". These block devices are provided by the untrusted environment and should be treated accordingly. We plan to provide transparent encryption on top of block devices in a future release."]}),"\n",(0,r.jsxs)(n.li,{children:[(0,r.jsx)(n.strong,{children:"Port forwarding"}),": This feature ",(0,r.jsx)(n.a,{href:"https://github.com/kata-containers/kata-containers/issues/1693",children:"isn't yet supported by Kata Containers"}),". You can ",(0,r.jsx)(n.a,{href:"https://docs.edgeless.systems/contrast/deployment#connect-to-the-contrast-coordinator",children:"deploy a port-forwarder"})," as a workaround."]}),"\n",(0,r.jsxs)(n.li,{children:[(0,r.jsx)(n.strong,{children:"Resource limits"}),": There is an existing bug on AKS where container memory limits are incorrectly applied. The current workaround involves using only memory requests instead of limits."]}),"\n"]}),"\n",(0,r.jsx)(n.h2,{id:"runtime-policies",children:"Runtime policies"}),"\n",(0,r.jsxs)(n.ul,{children:["\n",(0,r.jsxs)(n.li,{children:[(0,r.jsx)(n.strong,{children:"Coverage"}),": While the enforcement of workload policies generally functions well, ",(0,r.jsx)(n.a,{href:"https://github.com/microsoft/kata-containers/releases/tag/3.2.0.azl0.genpolicy",children:"there are scenarios not yet fully covered"}),". It's crucial to review deployments specifically for these edge cases."]}),"\n",(0,r.jsxs)(n.li,{children:[(0,r.jsx)(n.strong,{children:"Order of events"}),": The current policy evaluation mechanism on API requests isn't stateful, so it can't ensure a prescribed order of events. Consequently, there's no guaranteed enforcement that the ",(0,r.jsx)(n.a,{href:"/contrast/pr-preview/pr-994/0.8/components/service-mesh",children:"service mesh sidecar"})," container runs ",(0,r.jsx)(n.em,{children:"before"})," the workload container. This order ensures that all traffic between pods is securely encapsulated within TLS connections."]}),"\n",(0,r.jsxs)(n.li,{children:[(0,r.jsx)(n.strong,{children:"Absence of events"}),": Policies can't ensure certain events have happened. A container, such as the ",(0,r.jsx)(n.a,{href:"/contrast/pr-preview/pr-994/0.8/components/service-mesh",children:"service mesh sidecar"}),", can be omitted entirely. Environment variables may be missing."]}),"\n",(0,r.jsxs)(n.li,{children:[(0,r.jsx)(n.strong,{children:"Volume integrity checks"}),": While persistent volumes aren't supported yet, integrity checks don't currently cover other objects such as ",(0,r.jsx)(n.code,{children:"ConfigMaps"})," and ",(0,r.jsx)(n.code,{children:"Secrets"}),"."]}),"\n"]}),"\n",(0,r.jsx)(n.admonition,{type:"warning",children:(0,r.jsx)(n.p,{children:"The policy limitations, in particular the missing guarantee that our service mesh sidecar has been started before the workload container affects the service mesh implementation of Contrast. Currently, this requires inspecting the iptables rules on startup or terminating TLS connections in the workload directly."})}),"\n",(0,r.jsx)(n.h2,{id:"tooling-integration",children:"Tooling integration"}),"\n",(0,r.jsxs)(n.ul,{children:["\n",(0,r.jsxs)(n.li,{children:[(0,r.jsx)(n.strong,{children:"CLI availability"}),": The CLI tool is currently only available for Linux. This limitation arises because certain upstream dependencies haven't yet been ported to other platforms."]}),"\n"]}),"\n",(0,r.jsx)(n.h2,{id:"automatic-recovery-and-high-availability",children:"Automatic recovery and high availability"}),"\n",(0,r.jsx)(n.p,{children:"The Contrast Coordinator is a singleton and can't be scaled to more than one instance.\nWhen this instance's pod is restarted, for example for node maintenance, it needs to be recovered manually.\nIn a future release, we plan to support distributed Coordinator instances that can recover automatically."})]})}function h(e={}){const{wrapper:n}={...(0,s.R)(),...e.components};return n?(0,r.jsx)(n,{...e,children:(0,r.jsx)(d,{...e})}):d(e)}},28453:(e,n,t)=>{t.d(n,{R:()=>o,x:()=>a});var i=t(96540);const r={},s=i.createContext(r);function o(e){const n=i.useContext(s);return i.useMemo((function(){return"function"==typeof e?e(n):{...n,...e}}),[n,e])}function a(e){let n;return n=e.disableParentContext?"function"==typeof e.components?e.components(r):e.components||r:o(e.components),i.createElement(s.Provider,{value:n},e.children)}}}]);