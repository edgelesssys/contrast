"use strict";(self.webpackChunkcontrast_docs=self.webpackChunkcontrast_docs||[]).push([[4034],{28453:(e,n,r)=>{r.d(n,{R:()=>o,x:()=>i});var s=r(96540);const t={},a=s.createContext(t);function o(e){const n=s.useContext(a);return s.useMemo((function(){return"function"==typeof e?e(n):{...n,...e}}),[n,e])}function i(e){let n;return n=e.disableParentContext?"function"==typeof e.components?e.components(t):e.components||t:o(e.components),s.createElement(a.Provider,{value:n},e.children)}},74810:(e,n,r)=>{r.r(n),r.d(n,{assets:()=>c,contentTitle:()=>i,default:()=>d,frontMatter:()=>o,metadata:()=>s,toc:()=>l});const s=JSON.parse('{"id":"getting-started/cluster-setup","title":"Create a cluster","description":"Prerequisites","source":"@site/versioned_docs/version-1.4/getting-started/cluster-setup.md","sourceDirName":"getting-started","slug":"/getting-started/cluster-setup","permalink":"/contrast/pr-preview/pr-1396/1.4/getting-started/cluster-setup","draft":false,"unlisted":false,"editUrl":"https://github.com/edgelesssys/contrast/edit/main/docs/versioned_docs/version-1.4/getting-started/cluster-setup.md","tags":[],"version":"1.4","frontMatter":{},"sidebar":"docs","previous":{"title":"Install","permalink":"/contrast/pr-preview/pr-1396/1.4/getting-started/install"},"next":{"title":"Bare metal setup","permalink":"/contrast/pr-preview/pr-1396/1.4/getting-started/bare-metal"}}');var t=r(74848),a=r(28453);const o={},i="Create a cluster",c={},l=[{value:"Prerequisites",id:"prerequisites",level:2},{value:"Prepare using the AKS preview",id:"prepare-using-the-aks-preview",level:2},{value:"Create resource group",id:"create-resource-group",level:2},{value:"Create AKS cluster",id:"create-aks-cluster",level:2},{value:"Cleanup",id:"cleanup",level:2}];function u(e){const n={a:"a",code:"code",h1:"h1",h2:"h2",header:"header",li:"li",p:"p",pre:"pre",ul:"ul",...(0,a.R)(),...e.components};return(0,t.jsxs)(t.Fragment,{children:[(0,t.jsx)(n.header,{children:(0,t.jsx)(n.h1,{id:"create-a-cluster",children:"Create a cluster"})}),"\n",(0,t.jsx)(n.h2,{id:"prerequisites",children:"Prerequisites"}),"\n",(0,t.jsxs)(n.ul,{children:["\n",(0,t.jsxs)(n.li,{children:["Install version 2.44.1 or newer of the ",(0,t.jsx)(n.a,{href:"https://docs.microsoft.com/en-us/cli/azure/",children:"Azure CLI"}),". Note that your package manager will likely install an outdated version."]}),"\n",(0,t.jsxs)(n.li,{children:["Install a recent version of ",(0,t.jsx)(n.a,{href:"https://kubernetes.io/docs/tasks/tools/",children:"kubectl"}),"."]}),"\n"]}),"\n",(0,t.jsx)(n.h2,{id:"prepare-using-the-aks-preview",children:"Prepare using the AKS preview"}),"\n",(0,t.jsx)(n.p,{children:"First, log in to your Azure subscription:"}),"\n",(0,t.jsx)(n.pre,{children:(0,t.jsx)(n.code,{className:"language-bash",children:"az login\n"})}),"\n",(0,t.jsxs)(n.p,{children:["CoCo on AKS is currently in preview. An extension for the ",(0,t.jsx)(n.code,{children:"az"})," CLI is needed to create such a cluster.\nAdd the extension with the following commands:"]}),"\n",(0,t.jsx)(n.pre,{children:(0,t.jsx)(n.code,{className:"language-bash",children:"az extension add \\\n  --name aks-preview \\\n  --allow-preview true\naz extension update \\\n  --name aks-preview \\\n  --allow-preview true\n"})}),"\n",(0,t.jsx)(n.p,{children:"Then register the required feature flags in your subscription to allow access to the public preview:"}),"\n",(0,t.jsx)(n.pre,{children:(0,t.jsx)(n.code,{className:"language-bash",children:'az feature register \\\n    --namespace "Microsoft.ContainerService" \\\n    --name "KataCcIsolationPreview"\n'})}),"\n",(0,t.jsx)(n.p,{children:"Also enable the feature flag to disable SSH access to the AKS node (recommended, not required):"}),"\n",(0,t.jsx)(n.pre,{children:(0,t.jsx)(n.code,{className:"language-bash",children:'az feature register \\\n  --namespace "Microsoft.ContainerService" \\\n  --name "DisableSSHPreview"\n'})}),"\n",(0,t.jsxs)(n.p,{children:["The registration can take a few minutes. The status of the operation can be checked with the following\ncommand, which should show the registration state as ",(0,t.jsx)(n.code,{children:"Registered"}),":"]}),"\n",(0,t.jsx)(n.pre,{children:(0,t.jsx)(n.code,{className:"language-sh",children:'az feature show \\\n    --namespace "Microsoft.ContainerService" \\\n    --name "KataCcIsolationPreview" \\\n    --output table\naz feature show \\\n    --namespace "Microsoft.ContainerService" \\\n    --name "DisableSSHPreview" \\\n    --output table\n'})}),"\n",(0,t.jsx)(n.p,{children:"Afterward, refresh the registration of the ContainerService provider:"}),"\n",(0,t.jsx)(n.pre,{children:(0,t.jsx)(n.code,{className:"language-sh",children:'az provider register \\\n    --namespace "Microsoft.ContainerService"\n'})}),"\n",(0,t.jsx)(n.h2,{id:"create-resource-group",children:"Create resource group"}),"\n",(0,t.jsx)(n.p,{children:"The AKS with CoCo preview is currently available in the following locations:"}),"\n",(0,t.jsx)(n.pre,{children:(0,t.jsx)(n.code,{children:"CentralIndia\neastus\nEastUS2EUAP\nGermanyWestCentral\njapaneast\nnortheurope\nSwitzerlandNorth\nUAENorth\nwesteurope\nwestus\n"})}),"\n",(0,t.jsx)(n.p,{children:"Set the name of the resource group you want to use:"}),"\n",(0,t.jsx)(n.pre,{children:(0,t.jsx)(n.code,{className:"language-bash",children:'azResourceGroup="ContrastDemo"\n'})}),"\n",(0,t.jsx)(n.p,{children:"You can either use an existing one or create a new resource group with the following command:"}),"\n",(0,t.jsx)(n.pre,{children:(0,t.jsx)(n.code,{className:"language-bash",children:'azLocation="westus" # Select a location from the list above\n\naz group create \\\n  --name "${azResourceGroup:?}" \\\n  --location "${azLocation:?}"\n'})}),"\n",(0,t.jsx)(n.h2,{id:"create-aks-cluster",children:"Create AKS cluster"}),"\n",(0,t.jsx)(n.p,{children:"First, create a CoCo enabled AKS cluster with:"}),"\n",(0,t.jsx)(n.pre,{children:(0,t.jsx)(n.code,{className:"language-sh",children:'# Select the name for your AKS cluster\nazClusterName="ContrastDemo"\n\naz aks create \\\n  --resource-group "${azResourceGroup:?}" \\\n  --name "${azClusterName:?}" \\\n  --kubernetes-version 1.30 \\\n  --os-sku AzureLinux \\\n  --node-vm-size Standard_DC4as_cc_v5 \\\n  --workload-runtime KataCcIsolation \\\n  --node-count 1 \\\n  --ssh-access disabled\n'})}),"\n",(0,t.jsx)(n.p,{children:"Finally, update your kubeconfig with the credentials to access the cluster:"}),"\n",(0,t.jsx)(n.pre,{children:(0,t.jsx)(n.code,{className:"language-bash",children:'az aks get-credentials \\\n  --resource-group "${azResourceGroup:?}" \\\n  --name "${azClusterName:?}"\n'})}),"\n",(0,t.jsxs)(n.p,{children:["For validation, list the available nodes using ",(0,t.jsx)(n.code,{children:"kubectl"}),":"]}),"\n",(0,t.jsx)(n.pre,{children:(0,t.jsx)(n.code,{className:"language-bash",children:"kubectl get nodes\n"})}),"\n",(0,t.jsx)(n.p,{children:"It should show a single node:"}),"\n",(0,t.jsx)(n.pre,{children:(0,t.jsx)(n.code,{className:"language-bash",children:"NAME                                STATUS   ROLES    AGE     VERSION\naks-nodepool1-32049705-vmss000000   Ready    <none>   9m47s   v1.29.0\n"})}),"\n",(0,t.jsxs)(n.p,{children:["\ud83e\udd73 Congratulations. You're now ready to set up your first application with Contrast. Follow this ",(0,t.jsx)(n.a,{href:"/contrast/pr-preview/pr-1396/1.4/examples/emojivoto",children:"example"})," to learn how."]}),"\n",(0,t.jsx)(n.h2,{id:"cleanup",children:"Cleanup"}),"\n",(0,t.jsx)(n.p,{children:"After trying out Contrast, you might want to clean up the cloud resources created in this step.\nIn case you've created a new resource group, you can just delete that group with"}),"\n",(0,t.jsx)(n.pre,{children:(0,t.jsx)(n.code,{className:"language-sh",children:'az group delete \\\n  --name "${azResourceGroup:?}"\n'})}),"\n",(0,t.jsx)(n.p,{children:"Deleting the resource group will also delete the cluster and all other related resources."}),"\n",(0,t.jsx)(n.p,{children:"To only cleanup the AKS cluster and node pools, run"}),"\n",(0,t.jsx)(n.pre,{children:(0,t.jsx)(n.code,{className:"language-sh",children:'az aks delete \\\n  --resource-group "${azResourceGroup:?}" \\\n  --name "${azClusterName:?}"\n'})})]})}function d(e={}){const{wrapper:n}={...(0,a.R)(),...e.components};return n?(0,t.jsx)(n,{...e,children:(0,t.jsx)(u,{...e})}):u(e)}}}]);