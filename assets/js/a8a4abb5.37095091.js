"use strict";(self.webpackChunkcontrast_docs=self.webpackChunkcontrast_docs||[]).push([[8505],{92105:(e,n,t)=>{t.r(n),t.d(n,{assets:()=>o,contentTitle:()=>l,default:()=>h,frontMatter:()=>i,metadata:()=>s,toc:()=>d});const s=JSON.parse('{"id":"getting-started/bare-metal","title":"Prepare a bare-metal instance","description":"Hardware and firmware setup","source":"@site/versioned_docs/version-1.4/getting-started/bare-metal.md","sourceDirName":"getting-started","slug":"/getting-started/bare-metal","permalink":"/contrast/getting-started/bare-metal","draft":false,"unlisted":false,"editUrl":"https://github.com/edgelesssys/contrast/edit/main/docs/versioned_docs/version-1.4/getting-started/bare-metal.md","tags":[],"version":"1.4","frontMatter":{},"sidebar":"docs","previous":{"title":"Cluster setup","permalink":"/contrast/getting-started/cluster-setup"},"next":{"title":"Confidential emoji voting","permalink":"/contrast/examples/emojivoto"}}');var r=t(74848),a=t(28453);const i={},l="Prepare a bare-metal instance",o={},d=[{value:"Hardware and firmware setup",id:"hardware-and-firmware-setup",level:2},{value:"Kernel setup",id:"kernel-setup",level:2},{value:"K3s setup",id:"k3s-setup",level:2},{value:"Preparing a cluster for GPU usage",id:"preparing-a-cluster-for-gpu-usage",level:2}];function c(e){const n={a:"a",admonition:"admonition",code:"code",em:"em",h1:"h1",h2:"h2",header:"header",li:"li",ol:"ol",p:"p",pre:"pre",ul:"ul",...(0,a.R)(),...e.components},{TabItem:t,Tabs:s}=n;return t||u("TabItem",!0),s||u("Tabs",!0),(0,r.jsxs)(r.Fragment,{children:[(0,r.jsx)(n.header,{children:(0,r.jsx)(n.h1,{id:"prepare-a-bare-metal-instance",children:"Prepare a bare-metal instance"})}),"\n",(0,r.jsx)(n.h2,{id:"hardware-and-firmware-setup",children:"Hardware and firmware setup"}),"\n",(0,r.jsxs)(s,{queryString:"vendor",children:[(0,r.jsxs)(t,{value:"amd",label:"AMD SEV-SNP",children:[(0,r.jsxs)(n.ol,{children:["\n",(0,r.jsx)(n.li,{children:"Update your BIOS to a version that supports AMD SEV-SNP. Updating to the latest available version is recommended as newer versions will likely contain security patches for AMD SEV-SNP."}),"\n",(0,r.jsx)(n.li,{children:"Enter BIOS setup to enable SMEE, IOMMU, RMP coverage, and SEV-SNP. Set the SEV-ES ASID Space Limit to a non-zero number (higher is better)."}),"\n",(0,r.jsxs)(n.li,{children:["Download the latest firmware version for your processor from ",(0,r.jsx)(n.a,{href:"https://www.amd.com/de/developer/sev.html",children:"AMD"}),", unpack it, and place it in ",(0,r.jsx)(n.code,{children:"/lib/firmware/amd"}),"."]}),"\n"]}),(0,r.jsxs)(n.p,{children:["Consult AMD's ",(0,r.jsx)(n.a,{href:"https://www.amd.com/content/dam/amd/en/documents/epyc-technical-docs/tuning-guides/58207-using-sev-with-amd-epyc-processors.pdf",children:"Using SEV with AMD EPYC Processors user guide"})," for more information."]})]}),(0,r.jsx)(t,{value:"intel",label:"Intel TDX",children:(0,r.jsxs)(n.p,{children:["Follow Canonical's instructions on ",(0,r.jsx)(n.a,{href:"https://github.com/canonical/tdx?tab=readme-ov-file#43-enable-intel-tdx-in-the-hosts-bios",children:"setting up Intel TDX in the host's BIOS"}),"."]})})]}),"\n",(0,r.jsx)(n.h2,{id:"kernel-setup",children:"Kernel setup"}),"\n",(0,r.jsxs)(s,{queryString:"vendor",children:[(0,r.jsx)(t,{value:"amd",label:"AMD SEV-SNP",children:(0,r.jsx)(n.p,{children:"Install a kernel with version 6.11 or greater. If you're following this guide before 6.11 has been released, use 6.11-rc3. Don't use 6.11-rc4 - 6.11-rc6 as they contain a regression. 6.11-rc7+ might work."})}),(0,r.jsx)(t,{value:"intel",label:"Intel TDX",children:(0,r.jsxs)(n.p,{children:["Follow Canonical's instructions on ",(0,r.jsx)(n.a,{href:"https://github.com/canonical/tdx?tab=readme-ov-file#41-install-ubuntu-2404-server-image",children:"setting up Intel TDX on Ubuntu 24.04"}),". Note that Contrast currently only supports Intel TDX with Ubuntu 24.04."]})})]}),"\n",(0,r.jsxs)(n.p,{children:["Increase the ",(0,r.jsx)(n.code,{children:"user.max_inotify_instances"})," sysctl limit by adding ",(0,r.jsx)(n.code,{children:"user.max_inotify_instances=8192"})," to ",(0,r.jsx)(n.code,{children:"/etc/sysctl.d/99-sysctl.conf"})," and running ",(0,r.jsx)(n.code,{children:"sysctl --system"}),"."]}),"\n",(0,r.jsx)(n.h2,{id:"k3s-setup",children:"K3s setup"}),"\n",(0,r.jsxs)(n.ol,{children:["\n",(0,r.jsxs)(n.li,{children:["Follow the ",(0,r.jsx)(n.a,{href:"https://docs.k3s.io/",children:"K3s setup instructions"})," to create a cluster."]}),"\n",(0,r.jsxs)(n.li,{children:["Install a block storage provider such as ",(0,r.jsx)(n.a,{href:"https://docs.k3s.io/storage#setting-up-longhorn",children:"Longhorn"})," and mark it as the default storage class."]}),"\n"]}),"\n",(0,r.jsx)(n.h2,{id:"preparing-a-cluster-for-gpu-usage",children:"Preparing a cluster for GPU usage"}),"\n",(0,r.jsxs)(s,{queryString:"vendor",children:[(0,r.jsxs)(t,{value:"amd",label:"AMD SEV-SNP",children:[(0,r.jsxs)(n.p,{children:["To enable GPU usage on a Contrast cluster, some conditions need to be fulfilled for ",(0,r.jsx)(n.em,{children:"each cluster node"})," that should host GPU workloads:"]}),(0,r.jsxs)(n.ol,{children:["\n",(0,r.jsxs)(n.li,{children:["\n",(0,r.jsx)(n.p,{children:"You must activate the IOMMU. You can check by running:"}),"\n",(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-sh",children:"ls /sys/kernel/iommu_groups\n"})}),"\n",(0,r.jsxs)(n.p,{children:["If the output contains the group indices (",(0,r.jsx)(n.code,{children:"0"}),", ",(0,r.jsx)(n.code,{children:"1"}),", ...), the IOMMU is supported on the host.\nOtherwise, add ",(0,r.jsx)(n.code,{children:"intel_iommu=on"})," to the kernel command line."]}),"\n"]}),"\n",(0,r.jsxs)(n.li,{children:["\n",(0,r.jsx)(n.p,{children:"Additionally, the host kernel needs to have the following kernel configuration options enabled:"}),"\n",(0,r.jsxs)(n.ul,{children:["\n",(0,r.jsx)(n.li,{children:(0,r.jsx)(n.code,{children:"CONFIG_VFIO"})}),"\n",(0,r.jsx)(n.li,{children:(0,r.jsx)(n.code,{children:"CONFIG_VFIO_IOMMU_TYPE1"})}),"\n",(0,r.jsx)(n.li,{children:(0,r.jsx)(n.code,{children:"CONFIG_VFIO_MDEV"})}),"\n",(0,r.jsx)(n.li,{children:(0,r.jsx)(n.code,{children:"CONFIG_VFIO_MDEV_DEVICE"})}),"\n",(0,r.jsx)(n.li,{children:(0,r.jsx)(n.code,{children:"CONFIG_VFIO_PCI"})}),"\n"]}),"\n"]}),"\n"]}),(0,r.jsxs)(n.p,{children:["If the per-node requirements are fulfilled, deploy the ",(0,r.jsx)(n.a,{href:"https://docs.nvidia.com/datacenter/cloud-native/gpu-operator/latest",children:"NVIDIA GPU Operator"})," to the cluster. It provisions pod-VMs with GPUs via VFIO."]}),(0,r.jsxs)(n.p,{children:["Initially, label all nodes that ",(0,r.jsx)(n.em,{children:"should run GPU workloads"}),":"]}),(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-sh",children:"kubectl label node <node-name> nvidia.com/gpu.workload.config=vm-passthrough\n"})}),(0,r.jsx)(n.p,{children:"For a GPU-enabled Contrast cluster, you can then deploy the operator with the following command:"}),(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-sh",children:"helm install --wait --generate-name \\\n   -n gpu-operator --create-namespace \\\n   nvidia/gpu-operator \\\n   --version=v24.9.1 \\\n   --set sandboxWorkloads.enabled=true \\\n   --set sandboxWorkloads.defaultWorkload='vm-passthrough' \\\n   --set nfd.nodefeaturerules=true \\\n   --set vfioManager.enabled=true \\\n   --set kataManager.enabled=true \\\n   --set ccManager.enabled=true\n"})}),(0,r.jsxs)(n.p,{children:["Refer to the ",(0,r.jsx)(n.a,{href:"https://docs.nvidia.com/datacenter/cloud-native/gpu-operator/latest/getting-started.html",children:"official installation instructions"})," for details and further options."]}),(0,r.jsx)(n.p,{children:"Once the operator is deployed, check the available GPUs in the cluster:"}),(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-sh",children:'kubectl get nodes -l nvidia.com/gpu.present -o json | \\\n  jq \'.items[0].status.allocatable |\n    with_entries(select(.key | startswith("nvidia.com/"))) |\n    with_entries(select(.value != "0"))\'\n'})}),(0,r.jsx)(n.p,{children:"The above command should yield an output similar to the following, depending on what GPUs are available:"}),(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-json",children:'{\n   "nvidia.com/GH100_H100_PCIE": "1"\n}\n'})}),(0,r.jsxs)(n.p,{children:["These identifiers are then used to ",(0,r.jsx)(n.a,{href:"/contrast/deployment",children:"run GPU workloads on the cluster"}),"."]})]}),(0,r.jsx)(t,{value:"intel",label:"Intel TDX",children:(0,r.jsx)(n.admonition,{type:"warning",children:(0,r.jsx)(n.p,{children:"Currently, Contrast only supports GPU workloads on SEV-SNP-based clusters."})})})]})]})}function h(e={}){const{wrapper:n}={...(0,a.R)(),...e.components};return n?(0,r.jsx)(n,{...e,children:(0,r.jsx)(c,{...e})}):c(e)}function u(e,n){throw new Error("Expected "+(n?"component":"object")+" `"+e+"` to be defined: you likely forgot to import, pass, or provide it.")}},28453:(e,n,t)=>{t.d(n,{R:()=>i,x:()=>l});var s=t(96540);const r={},a=s.createContext(r);function i(e){const n=s.useContext(a);return s.useMemo((function(){return"function"==typeof e?e(n):{...n,...e}}),[n,e])}function l(e){let n;return n=e.disableParentContext?"function"==typeof e.components?e.components(r):e.components||r:i(e.components),s.createElement(a.Provider,{value:n},e.children)}}}]);