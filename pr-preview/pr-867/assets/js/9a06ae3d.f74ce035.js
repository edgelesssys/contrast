"use strict";(self.webpackChunkcontrast_docs=self.webpackChunkcontrast_docs||[]).push([[2912],{1629:(e,n,r)=>{r.r(n),r.d(n,{assets:()=>l,contentTitle:()=>o,default:()=>d,frontMatter:()=>a,metadata:()=>c,toc:()=>i});var s=r(4848),t=r(8453);const a={},o="Create a cluster",c={id:"getting-started/cluster-setup",title:"Create a cluster",description:"Prerequisites",source:"@site/docs/getting-started/cluster-setup.md",sourceDirName:"getting-started",slug:"/getting-started/cluster-setup",permalink:"/contrast/pr-preview/pr-867/next/getting-started/cluster-setup",draft:!1,unlisted:!1,editUrl:"https://github.com/edgelesssys/contrast/edit/main/docs/docs/getting-started/cluster-setup.md",tags:[],version:"current",frontMatter:{},sidebar:"docs",previous:{title:"Install",permalink:"/contrast/pr-preview/pr-867/next/getting-started/install"},next:{title:"Bare metal setup",permalink:"/contrast/pr-preview/pr-867/next/getting-started/bare-metal"}},l={},i=[{value:"Prerequisites",id:"prerequisites",level:2},{value:"Prepare using the AKS preview",id:"prepare-using-the-aks-preview",level:2},{value:"Create resource group",id:"create-resource-group",level:2},{value:"Create AKS cluster",id:"create-aks-cluster",level:2},{value:"Cleanup",id:"cleanup",level:2}];function u(e){const n={a:"a",code:"code",h1:"h1",h2:"h2",header:"header",p:"p",pre:"pre",...(0,t.R)(),...e.components};return(0,s.jsxs)(s.Fragment,{children:[(0,s.jsx)(n.header,{children:(0,s.jsx)(n.h1,{id:"create-a-cluster",children:"Create a cluster"})}),"\n",(0,s.jsx)(n.h2,{id:"prerequisites",children:"Prerequisites"}),"\n",(0,s.jsxs)(n.p,{children:["Install the latest version of the ",(0,s.jsx)(n.a,{href:"https://docs.microsoft.com/en-us/cli/azure/",children:"Azure CLI"}),"."]}),"\n",(0,s.jsxs)(n.p,{children:[(0,s.jsx)(n.a,{href:"https://docs.microsoft.com/en-us/cli/azure/authenticate-azure-cli",children:"Login to your account"}),", which needs\nto have the permissions to create an AKS cluster, by executing:"]}),"\n",(0,s.jsx)(n.pre,{children:(0,s.jsx)(n.code,{className:"language-bash",children:"az login\n"})}),"\n",(0,s.jsx)(n.h2,{id:"prepare-using-the-aks-preview",children:"Prepare using the AKS preview"}),"\n",(0,s.jsxs)(n.p,{children:["CoCo on AKS is currently in preview. An extension for the ",(0,s.jsx)(n.code,{children:"az"})," CLI is needed to create such a cluster.\nAdd the extension with the following commands:"]}),"\n",(0,s.jsx)(n.pre,{children:(0,s.jsx)(n.code,{className:"language-bash",children:"az extension add \\\n  --name aks-preview \\\n  --allow-preview true\naz extension update \\\n  --name aks-preview \\\n  --allow-preview true\n"})}),"\n",(0,s.jsx)(n.p,{children:"Then register the required feature flags in your subscription to allow access to the public preview:"}),"\n",(0,s.jsx)(n.pre,{children:(0,s.jsx)(n.code,{className:"language-bash",children:'az feature register \\\n    --namespace "Microsoft.ContainerService" \\\n    --name "KataCcIsolationPreview"\n'})}),"\n",(0,s.jsxs)(n.p,{children:["The registration can take a few minutes. The status of the operation can be checked with the following\ncommand, which should show the registration state as ",(0,s.jsx)(n.code,{children:"Registered"}),":"]}),"\n",(0,s.jsx)(n.pre,{children:(0,s.jsx)(n.code,{className:"language-sh",children:'az feature show \\\n    --namespace "Microsoft.ContainerService" \\\n    --name "KataCcIsolationPreview" \\\n    --output table\n'})}),"\n",(0,s.jsx)(n.p,{children:"Afterward, refresh the registration of the ContainerService provider:"}),"\n",(0,s.jsx)(n.pre,{children:(0,s.jsx)(n.code,{className:"language-sh",children:'az provider register \\\n    --namespace "Microsoft.ContainerService"\n'})}),"\n",(0,s.jsx)(n.h2,{id:"create-resource-group",children:"Create resource group"}),"\n",(0,s.jsx)(n.p,{children:"The AKS with CoCo preview is currently available in the following locations:"}),"\n",(0,s.jsx)(n.pre,{children:(0,s.jsx)(n.code,{children:"CentralIndia\neastus\nEastUS2EUAP\nGermanyWestCentral\njapaneast\nnortheurope\nSwitzerlandNorth\nUAENorth\nwesteurope\nwestus\n"})}),"\n",(0,s.jsx)(n.p,{children:"Set the name of the resource group you want to use:"}),"\n",(0,s.jsx)(n.pre,{children:(0,s.jsx)(n.code,{className:"language-bash",children:'azResourceGroup="ContrastDemo"\n'})}),"\n",(0,s.jsx)(n.p,{children:"You can either use an existing one or create a new resource group with the following command:"}),"\n",(0,s.jsx)(n.pre,{children:(0,s.jsx)(n.code,{className:"language-bash",children:'azLocation="westus" # Select a location from the list above\n\naz group create \\\n  --name "${azResourceGroup:?}" \\\n  --location "${azLocation:?}"\n'})}),"\n",(0,s.jsx)(n.h2,{id:"create-aks-cluster",children:"Create AKS cluster"}),"\n",(0,s.jsx)(n.p,{children:"First, we need to create an AKS cluster. We can't directly create a CoCo-enabled cluster, so we'll need to create a\nnon-CoCo cluster first, and then add a CoCo node pool, optionally replacing the non-CoCo node pool."}),"\n",(0,s.jsx)(n.p,{children:"We'll first start by creating the non-CoCo cluster:"}),"\n",(0,s.jsx)(n.pre,{children:(0,s.jsx)(n.code,{className:"language-sh",children:'# Select the name for your AKS cluster\nazClusterName="ContrastDemo"\n\naz aks create \\\n  --resource-group "${azResourceGroup:?}" \\\n  --name "${azClusterName:?}" \\\n  --kubernetes-version 1.29 \\\n  --os-sku AzureLinux \\\n  --node-vm-size Standard_DC4as_cc_v5 \\\n  --node-count 1 \\\n  --generate-ssh-keys\n'})}),"\n",(0,s.jsx)(n.p,{children:"We then add a second node pool with CoCo support:"}),"\n",(0,s.jsx)(n.pre,{children:(0,s.jsx)(n.code,{className:"language-bash",children:'az aks nodepool add \\\n  --resource-group "${azResourceGroup:?}" \\\n  --name nodepool2 \\\n  --cluster-name "${azClusterName:?}" \\\n  --node-count 1 \\\n  --os-sku AzureLinux \\\n  --node-vm-size Standard_DC4as_cc_v5 \\\n  --workload-runtime KataCcIsolation\n'})}),"\n",(0,s.jsx)(n.p,{children:"Optionally, we can now remove the non-CoCo node pool:"}),"\n",(0,s.jsx)(n.pre,{children:(0,s.jsx)(n.code,{className:"language-bash",children:'az aks nodepool delete \\\n  --resource-group "${azResourceGroup:?}" \\\n  --cluster-name "${azClusterName:?}" \\\n  --name nodepool1\n'})}),"\n",(0,s.jsx)(n.p,{children:"Finally, update your kubeconfig with the credentials to access the cluster:"}),"\n",(0,s.jsx)(n.pre,{children:(0,s.jsx)(n.code,{className:"language-bash",children:'az aks get-credentials \\\n  --resource-group "${azResourceGroup:?}" \\\n  --name "${azClusterName:?}"\n'})}),"\n",(0,s.jsx)(n.p,{children:"For validation, list the available nodes using kubectl:"}),"\n",(0,s.jsx)(n.pre,{children:(0,s.jsx)(n.code,{className:"language-bash",children:"kubectl get nodes\n"})}),"\n",(0,s.jsx)(n.p,{children:"It should show two nodes:"}),"\n",(0,s.jsx)(n.pre,{children:(0,s.jsx)(n.code,{className:"language-bash",children:"NAME                                STATUS   ROLES    AGE     VERSION\naks-nodepool1-32049705-vmss000000   Ready    <none>   9m47s   v1.29.0\naks-nodepool2-32238657-vmss000000   Ready    <none>   45s     v1.29.0\n"})}),"\n",(0,s.jsx)(n.h2,{id:"cleanup",children:"Cleanup"}),"\n",(0,s.jsx)(n.p,{children:"After trying out Contrast, you might want to clean up the cloud resources created in this step.\nIn case you've created a new resource group, you can just delete that group with"}),"\n",(0,s.jsx)(n.pre,{children:(0,s.jsx)(n.code,{className:"language-sh",children:'az group delete \\\n  --name "${azResourceGroup:?}"\n'})}),"\n",(0,s.jsx)(n.p,{children:"Deleting the resource group will also delete the cluster and all other related resources."}),"\n",(0,s.jsx)(n.p,{children:"To only cleanup the AKS cluster and node pools, run"}),"\n",(0,s.jsx)(n.pre,{children:(0,s.jsx)(n.code,{className:"language-sh",children:'az aks delete \\\n  --resource-group "${azResourceGroup:?}" \\\n  --name "${azClusterName:?}"\n'})})]})}function d(e={}){const{wrapper:n}={...(0,t.R)(),...e.components};return n?(0,s.jsx)(n,{...e,children:(0,s.jsx)(u,{...e})}):u(e)}},8453:(e,n,r)=>{r.d(n,{R:()=>o,x:()=>c});var s=r(6540);const t={},a=s.createContext(t);function o(e){const n=s.useContext(a);return s.useMemo((function(){return"function"==typeof e?e(n):{...n,...e}}),[n,e])}function c(e){let n;return n=e.disableParentContext?"function"==typeof e.components?e.components(t):e.components||t:o(e.components),s.createElement(a.Provider,{value:n},e.children)}}}]);