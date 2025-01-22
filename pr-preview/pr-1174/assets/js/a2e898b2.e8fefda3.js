"use strict";(self.webpackChunkcontrast_docs=self.webpackChunkcontrast_docs||[]).push([[678],{15365:(e,n,t)=>{t.r(n),t.d(n,{assets:()=>d,contentTitle:()=>a,default:()=>h,frontMatter:()=>o,metadata:()=>s,toc:()=>c});const s=JSON.parse('{"id":"components/runtime","title":"Contrast Runtime","description":"The Contrast runtime is responsible for starting pods as confidential virtual machines.","source":"@site/versioned_docs/version-1.3/components/runtime.md","sourceDirName":"components","slug":"/components/runtime","permalink":"/contrast/pr-preview/pr-1174/components/runtime","draft":false,"unlisted":false,"editUrl":"https://github.com/edgelesssys/contrast/edit/main/docs/versioned_docs/version-1.3/components/runtime.md","tags":[],"version":"1.3","frontMatter":{},"sidebar":"docs","previous":{"title":"Overview","permalink":"/contrast/pr-preview/pr-1174/components/overview"},"next":{"title":"Policies","permalink":"/contrast/pr-preview/pr-1174/components/policies"}}');var i=t(74848),r=t(28453);const o={},a="Contrast Runtime",d={},c=[{value:"Node-level components",id:"node-level-components",level:2},{value:"Containerd shim",id:"containerd-shim",level:3},{value:"Virtual machine manager (VMM)",id:"virtual-machine-manager-vmm",level:3},{value:"Snapshotters",id:"snapshotters",level:3},{value:"Pod-VM image",id:"pod-vm-image",level:3},{value:"Node installer DaemonSet",id:"node-installer-daemonset",level:2}];function l(e){const n={a:"a",code:"code",h1:"h1",h2:"h2",h3:"h3",header:"header",img:"img",li:"li",p:"p",pre:"pre",ul:"ul",...(0,r.R)(),...e.components};return(0,i.jsxs)(i.Fragment,{children:[(0,i.jsx)(n.header,{children:(0,i.jsx)(n.h1,{id:"contrast-runtime",children:"Contrast Runtime"})}),"\n",(0,i.jsxs)(n.p,{children:["The Contrast runtime is responsible for starting pods as confidential virtual machines.\nThis works by specifying the runtime class to be used in a pod spec and by registering the runtime class with the apiserver.\nThe ",(0,i.jsx)(n.code,{children:"RuntimeClass"})," resource defines a name for referencing the class and\na handler used by the container runtime (",(0,i.jsx)(n.code,{children:"containerd"}),") to identify the class."]}),"\n",(0,i.jsx)(n.pre,{children:(0,i.jsx)(n.code,{className:"language-yaml",children:"apiVersion: node.k8s.io/v1\nkind: RuntimeClass\nmetadata:\n  # This name is used by pods in the runtimeClassName field\n  name: contrast-cc-abcdef\n# This name is used by the\n# container runtime interface implementation (containerd)\nhandler: contrast-cc-abcdef\n"})}),"\n",(0,i.jsxs)(n.p,{children:["Confidential pods that are part of a Contrast deployment need to specify the\nsame runtime class in the ",(0,i.jsx)(n.code,{children:"runtimeClassName"})," field, so Kubernetes uses the\nContrast runtime instead of the default ",(0,i.jsx)(n.code,{children:"containerd"})," / ",(0,i.jsx)(n.code,{children:"runc"})," handler."]}),"\n",(0,i.jsx)(n.pre,{children:(0,i.jsx)(n.code,{className:"language-yaml",children:"apiVersion: v1\nkind: Pod\nspec:\n  runtimeClassName: contrast-cc-abcdef\n  # ...\n"})}),"\n",(0,i.jsx)(n.h2,{id:"node-level-components",children:"Node-level components"}),"\n",(0,i.jsxs)(n.p,{children:["The runtime consists of additional software components that need to be installed\nand configured on every SEV-SNP-enabled/TDX-enabled worker node.\nThis installation is performed automatically by the ",(0,i.jsxs)(n.a,{href:"#node-installer-daemonset",children:[(0,i.jsx)(n.code,{children:"node-installer"})," DaemonSet"]}),"."]}),"\n",(0,i.jsx)(n.p,{children:(0,i.jsx)(n.img,{alt:"Runtime components",src:t(11063).A+"",width:"3814",height:"1669"})}),"\n",(0,i.jsx)(n.h3,{id:"containerd-shim",children:"Containerd shim"}),"\n",(0,i.jsxs)(n.p,{children:["The ",(0,i.jsx)(n.code,{children:"handler"})," field in the Kubernetes ",(0,i.jsx)(n.code,{children:"RuntimeClass"})," instructs containerd not to use the default ",(0,i.jsx)(n.code,{children:"runc"})," implementation.\nInstead, containerd invokes a custom plugin called ",(0,i.jsx)(n.code,{children:"containerd-shim-contrast-cc-v2"}),".\nThis shim is described in more detail in the ",(0,i.jsx)(n.a,{href:"https://github.com/kata-containers/kata-containers/tree/3.4.0/src/runtime",children:"upstream source repository"})," and in the ",(0,i.jsx)(n.a,{href:"https://github.com/containerd/containerd/blob/main/core/runtime/v2/README.md",children:"containerd documentation"}),"."]}),"\n",(0,i.jsx)(n.h3,{id:"virtual-machine-manager-vmm",children:"Virtual machine manager (VMM)"}),"\n",(0,i.jsxs)(n.p,{children:["The ",(0,i.jsx)(n.code,{children:"containerd"})," shim uses a virtual machine monitor to create a confidential virtual machine for every pod.\nOn AKS, Contrast uses ",(0,i.jsx)(n.a,{href:"https://www.cloudhypervisor.org",children:(0,i.jsx)(n.code,{children:"cloud-hypervisor"})}),".\nOn bare metal, Contrast uses ",(0,i.jsx)(n.a,{href:"https://www.qemu.org/",children:(0,i.jsx)(n.code,{children:"QEMU"})}),".\nThe appropriate files are installed on every node by the ",(0,i.jsx)(n.a,{href:"#node-installer-daemonset",children:(0,i.jsx)(n.code,{children:"node-installer"})}),"."]}),"\n",(0,i.jsx)(n.h3,{id:"snapshotters",children:"Snapshotters"}),"\n",(0,i.jsxs)(n.p,{children:["Contrast uses ",(0,i.jsxs)(n.a,{href:"https://github.com/containerd/containerd/tree/v1.7.16/docs/snapshotters/README.md",children:[(0,i.jsx)(n.code,{children:"containerd"})," snapshotters"]})," to provide container images to the pod-VM.\nEach snapshotter consists of a host component that pulls container images and a guest component used to mount/pull container images."]}),"\n",(0,i.jsxs)(n.p,{children:["On AKS, Contrast uses the ",(0,i.jsx)(n.a,{href:"https://github.com/kata-containers/tardev-snapshotter",children:(0,i.jsx)(n.code,{children:"tardev"})})," snapshotter to provide container images as block devices to the pod-VM.\nThe ",(0,i.jsx)(n.code,{children:"tardev"})," snapshotter uses ",(0,i.jsx)(n.a,{href:"https://docs.kernel.org/admin-guide/device-mapper/verity.html",children:(0,i.jsx)(n.code,{children:"dm-verity"})})," to protect the integrity of container images.\nExpected ",(0,i.jsx)(n.code,{children:"dm-verity"})," container image hashes are part of Contrast runtime policies and are enforced by the kata-agent.\nThis enables workload attestation by specifying the allowed container image as part of the policy. Read ",(0,i.jsx)(n.a,{href:"/contrast/pr-preview/pr-1174/components/policies",children:"the chapter on policies"})," for more information."]}),"\n",(0,i.jsxs)(n.p,{children:["On bare metal, Contrast uses the ",(0,i.jsx)(n.a,{href:"https://github.com/containerd/nydus-snapshotter",children:(0,i.jsx)(n.code,{children:"nydus"})})," snapshotter to store metadata about the images. This metadata is communicated to the guest, so that it can pull the images itself."]}),"\n",(0,i.jsx)(n.h3,{id:"pod-vm-image",children:"Pod-VM image"}),"\n",(0,i.jsx)(n.p,{children:"Every pod-VM starts with the same guest image. It consists of an IGVM file and a root filesystem.\nThe IGVM file describes the initial memory contents of a pod-VM and consists of:"}),"\n",(0,i.jsxs)(n.ul,{children:["\n",(0,i.jsx)(n.li,{children:"Linux kernel image"}),"\n",(0,i.jsx)(n.li,{children:(0,i.jsx)(n.code,{children:"initrd"})}),"\n",(0,i.jsx)(n.li,{children:(0,i.jsx)(n.code,{children:"kernel commandline"})}),"\n"]}),"\n",(0,i.jsx)(n.p,{children:"Additionally, a root filesystem image is used that contains a read-only partition with the user space of the pod-VM and a verity partition to guarantee the integrity of the root filesystem.\nThe root filesystem contains systemd as the init system, and the kata agent for managing the pod."}),"\n",(0,i.jsx)(n.p,{children:"This pod-VM image isn't specific to any pod workload. Instead, container images are mounted at runtime."}),"\n",(0,i.jsx)(n.h2,{id:"node-installer-daemonset",children:"Node installer DaemonSet"}),"\n",(0,i.jsxs)(n.p,{children:["The ",(0,i.jsx)(n.code,{children:"RuntimeClass"})," resource above registers the runtime with the Kubernetes api.\nThe node-level installation is carried out by the Contrast node-installer\n",(0,i.jsx)(n.code,{children:"DaemonSet"})," that ships with every Contrast release."]}),"\n",(0,i.jsx)(n.p,{children:"After deploying the installer, it performs the following steps on each node:"}),"\n",(0,i.jsxs)(n.ul,{children:["\n",(0,i.jsxs)(n.li,{children:["Install the Contrast containerd shim (",(0,i.jsx)(n.code,{children:"containerd-shim-contrast-cc-v2"}),")"]}),"\n",(0,i.jsxs)(n.li,{children:["Install ",(0,i.jsx)(n.code,{children:"cloud-hypervisor"})," or ",(0,i.jsx)(n.code,{children:"QEMU"})," as the virtual machine manager (VMM)"]}),"\n",(0,i.jsx)(n.li,{children:"Install an IGVM file or separate firmware and kernel files for pod-VMs of this class"}),"\n",(0,i.jsx)(n.li,{children:"Install a read only root filesystem disk image for the pod-VMs of this class"}),"\n",(0,i.jsxs)(n.li,{children:["Reconfigure ",(0,i.jsx)(n.code,{children:"containerd"})," by adding a runtime plugin that corresponds to the ",(0,i.jsx)(n.code,{children:"handler"})," field of the Kubernetes ",(0,i.jsx)(n.code,{children:"RuntimeClass"})]}),"\n",(0,i.jsxs)(n.li,{children:["Restart ",(0,i.jsx)(n.code,{children:"containerd"})," to make it aware of the new plugin"]}),"\n"]})]})}function h(e={}){const{wrapper:n}={...(0,r.R)(),...e.components};return n?(0,i.jsx)(n,{...e,children:(0,i.jsx)(l,{...e})}):l(e)}},11063:(e,n,t)=>{t.d(n,{A:()=>s});const s=t.p+"assets/images/runtime-c41c29928f42a90474f03213074a5f97.svg"},28453:(e,n,t)=>{t.d(n,{R:()=>o,x:()=>a});var s=t(96540);const i={},r=s.createContext(i);function o(e){const n=s.useContext(r);return s.useMemo((function(){return"function"==typeof e?e(n):{...n,...e}}),[n,e])}function a(e){let n;return n=e.disableParentContext?"function"==typeof e.components?e.components(i):e.components||i:o(e.components),s.createElement(r.Provider,{value:n},e.children)}}}]);