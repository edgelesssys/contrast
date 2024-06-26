"use strict";(self.webpackChunkcontrast_docs=self.webpackChunkcontrast_docs||[]).push([[6408],{847:(e,t,n)=>{n.r(t),n.d(t,{assets:()=>c,contentTitle:()=>o,default:()=>h,frontMatter:()=>r,metadata:()=>a,toc:()=>d});var s=n(4848),i=n(8453);const r={slug:"/",id:"intro"},o="Contrast",a={id:"intro",title:"Contrast",description:"Welcome to the documentation of Contrast! Contrast runs confidential container deployments on Kubernetes at scale.",source:"@site/versioned_docs/version-0.5/intro.md",sourceDirName:".",slug:"/",permalink:"/contrast/pr-preview/pr-640/0.5/",draft:!1,unlisted:!1,editUrl:"https://github.com/edgelesssys/contrast/edit/main/docs/versioned_docs/version-0.5/intro.md",tags:[],version:"0.5",frontMatter:{slug:"/",id:"intro"},sidebar:"docs",next:{title:"Confidential Containers",permalink:"/contrast/pr-preview/pr-640/0.5/basics/confidential-containers"}},c={},d=[{value:"Goal",id:"goal",level:2},{value:"Use Cases",id:"use-cases",level:2},{value:"Next steps",id:"next-steps",level:2}];function l(e){const t={a:"a",admonition:"admonition",em:"em",h1:"h1",h2:"h2",li:"li",p:"p",ul:"ul",...(0,i.R)(),...e.components};return(0,s.jsxs)(s.Fragment,{children:[(0,s.jsx)(t.h1,{id:"contrast",children:"Contrast"}),"\n",(0,s.jsx)(t.p,{children:"Welcome to the documentation of Contrast! Contrast runs confidential container deployments on Kubernetes at scale."}),"\n",(0,s.jsxs)(t.p,{children:["Contrast is based on the ",(0,s.jsx)(t.a,{href:"https://github.com/kata-containers/kata-containers",children:"Kata Containers"})," and\n",(0,s.jsx)(t.a,{href:"https://github.com/confidential-containers",children:"Confidential Containers"})," projects.\nConfidential Containers are Kubernetes pods that are executed inside a confidential micro-VM and provide strong hardware-based isolation from the surrounding environment.\nThis works with unmodified containers in a lift-and-shift approach.\nContrast currently targets the ",(0,s.jsx)(t.a,{href:"https://learn.microsoft.com/en-us/azure/confidential-computing/confidential-containers-on-aks-preview",children:"CoCo preview on AKS"}),"."]}),"\n",(0,s.jsx)(t.admonition,{type:"tip",children:(0,s.jsxs)(t.p,{children:["See the \ud83d\udcc4",(0,s.jsx)(t.a,{href:"https://content.edgeless.systems/hubfs/Confidential%20Computing%20Whitepaper.pdf",children:"whitepaper"})," for more information on confidential computing."]})}),"\n",(0,s.jsx)(t.h2,{id:"goal",children:"Goal"}),"\n",(0,s.jsx)(t.p,{children:"Contrast is designed to keep all data always encrypted and to prevent access from the infrastructure layer. It removes the infrastructure provider from the trusted computing base (TCB). This includes access from datacenter employees, privileged cloud admins, own cluster administrators, and attackers coming through the infrastructure, for example, malicious co-tenants escalating their privileges."}),"\n",(0,s.jsx)(t.p,{children:"Contrast integrates fluently with the existing Kubernetes workflows. It's compatible with managed Kubernetes, can be installed as a day-2 operation and imposes only minimal changes to your deployment flow."}),"\n",(0,s.jsx)(t.h2,{id:"use-cases",children:"Use Cases"}),"\n",(0,s.jsxs)(t.p,{children:["Contrast provides unique security ",(0,s.jsx)(t.a,{href:"/contrast/pr-preview/pr-640/0.5/basics/features",children:"features"})," and ",(0,s.jsx)(t.a,{href:"/contrast/pr-preview/pr-640/0.5/basics/security-benefits",children:"benefits"}),". The core use cases are:"]}),"\n",(0,s.jsxs)(t.ul,{children:["\n",(0,s.jsx)(t.li,{children:"Increasing the security of your containers"}),"\n",(0,s.jsx)(t.li,{children:"Moving sensitive workloads from on-prem to the cloud with Confidential Computing"}),"\n",(0,s.jsx)(t.li,{children:"Shielding the code and data even from the own cluster administrators"}),"\n",(0,s.jsx)(t.li,{children:"Increasing the trustworthiness of your SaaS offerings"}),"\n",(0,s.jsx)(t.li,{children:"Simplifying regulatory compliance"}),"\n",(0,s.jsx)(t.li,{children:"Multi-party computation for data collaboration"}),"\n"]}),"\n",(0,s.jsx)(t.h2,{id:"next-steps",children:"Next steps"}),"\n",(0,s.jsxs)(t.p,{children:["You can learn more about the concept of ",(0,s.jsx)(t.a,{href:"/contrast/pr-preview/pr-640/0.5/basics/confidential-containers",children:"Confidential Containers"}),", ",(0,s.jsx)(t.a,{href:"/contrast/pr-preview/pr-640/0.5/basics/features",children:"features"}),", and ",(0,s.jsx)(t.a,{href:"/contrast/pr-preview/pr-640/0.5/basics/security-benefits",children:"security benefits"})," of Contrast in this section. To jump right into the action head to ",(0,s.jsx)(t.a,{href:"getting-started",children:(0,s.jsx)(t.em,{children:"Getting started"})}),"."]})]})}function h(e={}){const{wrapper:t}={...(0,i.R)(),...e.components};return t?(0,s.jsx)(t,{...e,children:(0,s.jsx)(l,{...e})}):l(e)}},8453:(e,t,n)=>{n.d(t,{R:()=>o,x:()=>a});var s=n(6540);const i={},r=s.createContext(i);function o(e){const t=s.useContext(r);return s.useMemo((function(){return"function"==typeof e?e(t):{...t,...e}}),[t,e])}function a(e){let t;return t=e.disableParentContext?"function"==typeof e.components?e.components(i):e.components||i:o(e.components),s.createElement(r.Provider,{value:t},e.children)}}}]);