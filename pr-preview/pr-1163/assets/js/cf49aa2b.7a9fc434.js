"use strict";(self.webpackChunkcontrast_docs=self.webpackChunkcontrast_docs||[]).push([[4703],{54405:(e,t,n)=>{n.r(t),n.d(t,{assets:()=>l,contentTitle:()=>o,default:()=>h,frontMatter:()=>i,metadata:()=>r,toc:()=>c});const r=JSON.parse('{"id":"getting-started/bare-metal","title":"Prepare a bare-metal instance","description":"Hardware and firmware setup","source":"@site/docs/getting-started/bare-metal.md","sourceDirName":"getting-started","slug":"/getting-started/bare-metal","permalink":"/contrast/pr-preview/pr-1163/next/getting-started/bare-metal","draft":false,"unlisted":false,"editUrl":"https://github.com/edgelesssys/contrast/edit/main/docs/docs/getting-started/bare-metal.md","tags":[],"version":"current","frontMatter":{},"sidebar":"docs","previous":{"title":"Cluster setup","permalink":"/contrast/pr-preview/pr-1163/next/getting-started/cluster-setup"},"next":{"title":"Confidential emoji voting","permalink":"/contrast/pr-preview/pr-1163/next/examples/emojivoto"}}');var s=n(74848),a=n(28453);const i={},o="Prepare a bare-metal instance",l={},c=[{value:"Hardware and firmware setup",id:"hardware-and-firmware-setup",level:2},{value:"Kernel Setup",id:"kernel-setup",level:2},{value:"K3s Setup",id:"k3s-setup",level:2}];function d(e){const t={a:"a",code:"code",h1:"h1",h2:"h2",header:"header",li:"li",ol:"ol",p:"p",...(0,a.R)(),...e.components},{TabItem:n,Tabs:r}=t;return n||u("TabItem",!0),r||u("Tabs",!0),(0,s.jsxs)(s.Fragment,{children:[(0,s.jsx)(t.header,{children:(0,s.jsx)(t.h1,{id:"prepare-a-bare-metal-instance",children:"Prepare a bare-metal instance"})}),"\n",(0,s.jsx)(t.h2,{id:"hardware-and-firmware-setup",children:"Hardware and firmware setup"}),"\n",(0,s.jsxs)(r,{queryString:"vendor",children:[(0,s.jsxs)(n,{value:"amd",label:"AMD SEV-SNP",children:[(0,s.jsxs)(t.ol,{children:["\n",(0,s.jsx)(t.li,{children:"Update your BIOS to a version that supports AMD SEV-SNP. Updating to the latest available version is recommended as newer versions will likely contain security patches for AMD SEV-SNP."}),"\n",(0,s.jsx)(t.li,{children:"Enter BIOS setup to enable SMEE, IOMMU, RMP coverage, and SEV-SNP. Set the SEV-ES ASID Space Limit to a non-zero number (higher is better)."}),"\n",(0,s.jsxs)(t.li,{children:["Download the latest firmware version for your processor from ",(0,s.jsx)(t.a,{href:"https://www.amd.com/de/developer/sev.html",children:"AMD"}),", unpack it, and place it in ",(0,s.jsx)(t.code,{children:"/lib/firmware/amd"}),"."]}),"\n"]}),(0,s.jsxs)(t.p,{children:["Consult AMD's ",(0,s.jsx)(t.a,{href:"https://www.amd.com/content/dam/amd/en/documents/epyc-technical-docs/tuning-guides/58207-using-sev-with-amd-epyc-processors.pdf",children:"Using SEV with AMD EPYC Processors user guide"})," for more information."]})]}),(0,s.jsx)(n,{value:"intel",label:"Intel TDX",children:(0,s.jsxs)(t.p,{children:["Follow Canonical's instructions on ",(0,s.jsx)(t.a,{href:"https://github.com/canonical/tdx?tab=readme-ov-file#43-enable-intel-tdx-in-the-hosts-bios",children:"setting up Intel TDX in the host's BIOS"}),"."]})})]}),"\n",(0,s.jsx)(t.h2,{id:"kernel-setup",children:"Kernel Setup"}),"\n",(0,s.jsxs)(r,{queryString:"vendor",children:[(0,s.jsx)(n,{value:"amd",label:"AMD SEV-SNP",children:(0,s.jsx)(t.p,{children:"Install a kernel with version 6.11 or greater. If you're following this guide before 6.11 has been released, use 6.11-rc3. Don't use 6.11-rc4 - 6.11-rc6 as they contain a regression. 6.11-rc7+ might work."})}),(0,s.jsx)(n,{value:"intel",label:"Intel TDX",children:(0,s.jsxs)(t.p,{children:["Follow Canonical's instructions on ",(0,s.jsx)(t.a,{href:"https://github.com/canonical/tdx?tab=readme-ov-file#41-install-ubuntu-2404-server-image",children:"setting up Intel TDX on Ubuntu 24.04"}),". Note that Contrast currently only supports Intel TDX with Ubuntu 24.04."]})})]}),"\n",(0,s.jsxs)(t.p,{children:["Increase the ",(0,s.jsx)(t.code,{children:"user.max_inotify_instances"})," sysctl limit by adding ",(0,s.jsx)(t.code,{children:"user.max_inotify_instances=8192"})," to ",(0,s.jsx)(t.code,{children:"/etc/sysctl.d/99-sysctl.conf"})," and running ",(0,s.jsx)(t.code,{children:"sysctl --system"}),"."]}),"\n",(0,s.jsx)(t.h2,{id:"k3s-setup",children:"K3s Setup"}),"\n",(0,s.jsxs)(t.ol,{children:["\n",(0,s.jsxs)(t.li,{children:["Follow the ",(0,s.jsx)(t.a,{href:"https://docs.k3s.io/",children:"K3s setup instructions"})," to create a cluster."]}),"\n",(0,s.jsxs)(t.li,{children:["Install a block storage provider such as ",(0,s.jsx)(t.a,{href:"https://docs.k3s.io/storage#setting-up-longhorn",children:"Longhorn"})," and mark it as the default storage class."]}),"\n"]})]})}function h(e={}){const{wrapper:t}={...(0,a.R)(),...e.components};return t?(0,s.jsx)(t,{...e,children:(0,s.jsx)(d,{...e})}):d(e)}function u(e,t){throw new Error("Expected "+(t?"component":"object")+" `"+e+"` to be defined: you likely forgot to import, pass, or provide it.")}},28453:(e,t,n)=>{n.d(t,{R:()=>i,x:()=>o});var r=n(96540);const s={},a=r.createContext(s);function i(e){const t=r.useContext(a);return r.useMemo((function(){return"function"==typeof e?e(t):{...t,...e}}),[t,e])}function o(e){let t;return t=e.disableParentContext?"function"==typeof e.components?e.components(s):e.components||s:i(e.components),r.createElement(a.Provider,{value:t},e.children)}}}]);