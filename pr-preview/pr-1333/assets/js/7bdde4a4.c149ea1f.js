"use strict";(self.webpackChunkcontrast_docs=self.webpackChunkcontrast_docs||[]).push([[3752],{68666:(e,t,n)=>{n.r(t),n.d(t,{assets:()=>l,contentTitle:()=>a,default:()=>h,frontMatter:()=>i,metadata:()=>r,toc:()=>d});const r=JSON.parse('{"id":"examples/mysql","title":"Encrypted volume mount","description":"This tutorial guides you through deploying a simple application with an","source":"@site/docs/examples/mysql.md","sourceDirName":"examples","slug":"/examples/mysql","permalink":"/contrast/pr-preview/pr-1333/next/examples/mysql","draft":false,"unlisted":false,"editUrl":"https://github.com/edgelesssys/contrast/edit/main/docs/docs/examples/mysql.md","tags":[],"version":"current","frontMatter":{},"sidebar":"docs","previous":{"title":"Confidential emoji voting","permalink":"/contrast/pr-preview/pr-1333/next/examples/emojivoto"},"next":{"title":"Workload deployment","permalink":"/contrast/pr-preview/pr-1333/next/deployment"}}');var o=n(74848),s=n(28453);const i={},a="Encrypted volume mount",l={},d=[{value:"Prerequisites",id:"prerequisites",level:2},{value:"Steps to deploy MySQL with Contrast",id:"steps-to-deploy-mysql-with-contrast",level:2},{value:"Download the deployment files",id:"download-the-deployment-files",level:3},{value:"Deploy the Contrast runtime",id:"deploy-the-contrast-runtime",level:3},{value:"Download the Contrast Coordinator resource",id:"download-the-contrast-coordinator-resource",level:3},{value:"Generate policy annotations and manifest",id:"generate-policy-annotations-and-manifest",level:3},{value:"Deploy the Coordinator",id:"deploy-the-coordinator",level:3},{value:"Set the manifest",id:"set-the-manifest",level:3},{value:"Deploy MySQL",id:"deploy-mysql",level:3},{value:"Verifying the deployment as a user",id:"verifying-the-deployment-as-a-user",level:2},{value:"Verifying the Coordinator",id:"verifying-the-coordinator",level:3},{value:"Auditing the manifest history and artifacts",id:"auditing-the-manifest-history-and-artifacts",level:3},{value:"Connecting to the application",id:"connecting-to-the-application",level:3},{value:"Updating the deployment",id:"updating-the-deployment",level:2}];function c(e){const t={a:"a",admonition:"admonition",code:"code",h1:"h1",h2:"h2",h3:"h3",header:"header",li:"li",p:"p",pre:"pre",strong:"strong",ul:"ul",...(0,s.R)(),...e.components},{TabItem:n,Tabs:r}=t;return n||p("TabItem",!0),r||p("Tabs",!0),(0,o.jsxs)(o.Fragment,{children:[(0,o.jsx)(t.header,{children:(0,o.jsx)(t.h1,{id:"encrypted-volume-mount",children:"Encrypted volume mount"})}),"\n",(0,o.jsx)(t.p,{children:(0,o.jsxs)(t.strong,{children:["This tutorial guides you through deploying a simple application with an\nencrypted MySQL database using the Contrast ",(0,o.jsx)(t.a,{href:"/contrast/pr-preview/pr-1333/next/architecture/secrets#workload-secrets",children:"workload\nsecret"}),"."]})}),"\n",(0,o.jsxs)(t.p,{children:[(0,o.jsx)(t.a,{href:"https://mysql.com",children:"MySQL"})," is an open-source database used to organize data into\ntables and quickly retrieve information about its content. All of the data in a\nMySQL database is stored in the ",(0,o.jsx)(t.code,{children:"/var/lib/mysql"})," directory. In this example, we\nuse the workload secret to setup an encrypted LUKS mount for the\n",(0,o.jsx)(t.code,{children:"/var/lib/mysql"})," directory to easily deploy an application with encrypted\npersistent storage using Contrast."]}),"\n",(0,o.jsxs)(t.p,{children:["The resources provided in this demo are designed for educational purposes and\nshouldn't be used in a production environment without proper evaluation. When\nworking with persistent storage, regular backups are recommended in order to\nprevent data loss. For confidential applications, please also refer to the\n",(0,o.jsx)(t.a,{href:"/contrast/pr-preview/pr-1333/next/architecture/security-considerations",children:"security considerations"}),". Also be\naware of the differences in security implications of the workload secrets for\nthe data owner and the workload owner. For more details, see the ",(0,o.jsx)(t.a,{href:"/contrast/pr-preview/pr-1333/next/architecture/secrets#workload-secrets",children:"Workload\nSecrets"})," documentation."]}),"\n",(0,o.jsx)(t.h2,{id:"prerequisites",children:"Prerequisites"}),"\n",(0,o.jsxs)(t.ul,{children:["\n",(0,o.jsx)(t.li,{children:"Installed Contrast CLI"}),"\n",(0,o.jsxs)(t.li,{children:["A running Kubernetes cluster with support for confidential containers, either on ",(0,o.jsx)(t.a,{href:"/contrast/pr-preview/pr-1333/next/getting-started/cluster-setup",children:"AKS"})," or on ",(0,o.jsx)(t.a,{href:"/contrast/pr-preview/pr-1333/next/getting-started/bare-metal",children:"bare metal"})]}),"\n"]}),"\n",(0,o.jsx)(t.h2,{id:"steps-to-deploy-mysql-with-contrast",children:"Steps to deploy MySQL with Contrast"}),"\n",(0,o.jsx)(t.h3,{id:"download-the-deployment-files",children:"Download the deployment files"}),"\n",(0,o.jsx)(t.p,{children:"The MySQL deployment files are part of the Contrast release. You can download them by running:"}),"\n",(0,o.jsx)(t.pre,{children:(0,o.jsx)(t.code,{className:"language-sh",children:"curl -fLO https://github.com/edgelesssys/contrast/releases/latest/download/mysql-demo.yml --create-dirs --output-dir deployment\n"})}),"\n",(0,o.jsx)(t.h3,{id:"deploy-the-contrast-runtime",children:"Deploy the Contrast runtime"}),"\n",(0,o.jsxs)(t.p,{children:["Contrast depends on a ",(0,o.jsxs)(t.a,{href:"/contrast/pr-preview/pr-1333/next/components/runtime",children:["custom Kubernetes ",(0,o.jsx)(t.code,{children:"RuntimeClass"})]}),",\nwhich needs to be installed to the cluster initially.\nThis consists of a ",(0,o.jsx)(t.code,{children:"RuntimeClass"})," resource and a ",(0,o.jsx)(t.code,{children:"DaemonSet"})," that performs installation on worker nodes.\nThis step is only required once for each version of the runtime.\nIt can be shared between Contrast deployments."]}),"\n",(0,o.jsxs)(r,{queryString:"platform",children:[(0,o.jsx)(n,{value:"aks-clh-snp",label:"AKS",default:!0,children:(0,o.jsx)(t.pre,{children:(0,o.jsx)(t.code,{className:"language-sh",children:"kubectl apply -f https://github.com/edgelesssys/contrast/releases/latest/download/runtime-aks-clh-snp.yml\n"})})}),(0,o.jsx)(n,{value:"k3s-qemu-snp",label:"Bare metal (SEV-SNP)",children:(0,o.jsx)(t.pre,{children:(0,o.jsx)(t.code,{className:"language-sh",children:"kubectl apply -f https://github.com/edgelesssys/contrast/releases/latest/download/runtime-k3s-qemu-snp.yml\n"})})}),(0,o.jsx)(n,{value:"k3s-qemu-tdx",label:"Bare metal (TDX)",children:(0,o.jsx)(t.pre,{children:(0,o.jsx)(t.code,{className:"language-sh",children:"kubectl apply -f https://github.com/edgelesssys/contrast/releases/latest/download/runtime-k3s-qemu-tdx.yml\n"})})})]}),"\n",(0,o.jsx)(t.h3,{id:"download-the-contrast-coordinator-resource",children:"Download the Contrast Coordinator resource"}),"\n",(0,o.jsx)(t.p,{children:"Download the Kubernetes resource of the Contrast Coordinator, comprising a single replica deployment and a\nLoadBalancer service. Put it next to your resources:"}),"\n",(0,o.jsx)(t.pre,{children:(0,o.jsx)(t.code,{className:"language-sh",children:"curl -fLO https://github.com/edgelesssys/contrast/releases/latest/download/coordinator.yml --output-dir deployment\n"})}),"\n",(0,o.jsx)(t.h3,{id:"generate-policy-annotations-and-manifest",children:"Generate policy annotations and manifest"}),"\n",(0,o.jsxs)(t.p,{children:["Run the ",(0,o.jsx)(t.code,{children:"generate"})," command to generate the execution policies and add them as\nannotations to your deployment files. A ",(0,o.jsx)(t.code,{children:"manifest.json"})," file with the reference values\nof your deployment will be created:"]}),"\n",(0,o.jsxs)(r,{queryString:"platform",children:[(0,o.jsx)(n,{value:"aks-clh-snp",label:"AKS",default:!0,children:(0,o.jsx)(t.pre,{children:(0,o.jsx)(t.code,{className:"language-sh",children:"contrast generate --reference-values aks-clh-snp deployment/\n"})})}),(0,o.jsxs)(n,{value:"k3s-qemu-snp",label:"Bare metal (SEV-SNP)",children:[(0,o.jsx)(t.pre,{children:(0,o.jsx)(t.code,{className:"language-sh",children:"contrast generate --reference-values k3s-qemu-snp deployment/\n"})}),(0,o.jsx)(t.admonition,{title:"Missing TCB values",type:"note",children:(0,o.jsxs)(t.p,{children:["On bare-metal SEV-SNP, ",(0,o.jsx)(t.code,{children:"contrast generate"})," is unable to fill in the ",(0,o.jsx)(t.code,{children:"MinimumTCB"})," values as they can vary between platforms.\nThey will have to be filled in manually.\nIf you don't know the correct values use ",(0,o.jsx)(t.code,{children:'{"BootloaderVersion":255,"TEEVersion":255,"SNPVersion":255,"MicrocodeVersion":255}'})," and observe the real values in the error messages in the following steps. This should only be done in a secure environment. Note that the values will differ between CPU models."]})})]}),(0,o.jsxs)(n,{value:"k3s-qemu-tdx",label:"Bare metal (TDX)",children:[(0,o.jsx)(t.pre,{children:(0,o.jsx)(t.code,{className:"language-sh",children:"contrast generate --reference-values k3s-qemu-tdx deployment/\n"})}),(0,o.jsx)(t.admonition,{title:"Missing TCB values",type:"note",children:(0,o.jsxs)(t.p,{children:["On bare-metal TDX, ",(0,o.jsx)(t.code,{children:"contrast generate"})," is unable to fill in the ",(0,o.jsx)(t.code,{children:"MinimumTeeTcbSvn"})," and ",(0,o.jsx)(t.code,{children:"MrSeam"})," TCB values as they can vary between platforms.\nThey will have to be filled in manually.\nIf you don't know the correct values use ",(0,o.jsx)(t.code,{children:"ffffffffffffffffffffffffffffffff"})," and ",(0,o.jsx)(t.code,{children:"000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"})," respectively and observe the real values in the error messages in the following steps. This should only be done in a secure environment."]})})]})]}),"\n",(0,o.jsxs)(t.admonition,{title:"Runtime class and Initializer",type:"note",children:[(0,o.jsxs)(t.p,{children:["The deployment YAML shipped for this demo is already configured to be used with Contrast.\nA ",(0,o.jsx)(t.a,{href:"../components/runtime",children:"runtime class"})," ",(0,o.jsx)(t.code,{children:"contrast-cc"}),"\nwas added to the pods to signal they should be run as Confidential Containers. During the generation process,\nthe Contrast ",(0,o.jsx)(t.a,{href:"/contrast/pr-preview/pr-1333/next/components/overview#the-initializer",children:"Initializer"})," will be added as an init container to these\nworkloads. It will attest the pod to the Coordinator and fetch the workload certificates and the workload secret."]}),(0,o.jsxs)(t.p,{children:["Further, the deployment YAML is also configured with the Contrast ",(0,o.jsx)(t.a,{href:"/contrast/pr-preview/pr-1333/next/components/service-mesh",children:"service mesh"}),".\nThe configured service mesh proxy provides transparent protection for the communication between\nthe MySQL server and client."]})]}),"\n",(0,o.jsx)(t.h3,{id:"deploy-the-coordinator",children:"Deploy the Coordinator"}),"\n",(0,o.jsx)(t.p,{children:"Deploy the Coordinator resource first by applying its resource definition:"}),"\n",(0,o.jsx)(t.pre,{children:(0,o.jsx)(t.code,{className:"language-sh",children:"kubectl apply -f deployment/coordinator.yml\n"})}),"\n",(0,o.jsx)(t.h3,{id:"set-the-manifest",children:"Set the manifest"}),"\n",(0,o.jsx)(t.p,{children:"Configure the Coordinator with a manifest. It might take up to a few minutes\nfor the load balancer to be created and the Coordinator being available."}),"\n",(0,o.jsx)(t.pre,{children:(0,o.jsx)(t.code,{className:"language-sh",children:'coordinator=$(kubectl get svc coordinator -o=jsonpath=\'{.status.loadBalancer.ingress[0].ip}\')\necho "The user API of your Contrast Coordinator is available at $coordinator:1313"\ncontrast set -c "${coordinator}:1313" deployment/\n'})}),"\n",(0,o.jsx)(t.p,{children:"The CLI will use the reference values from the manifest to attest the Coordinator deployment\nduring the TLS handshake. If the connection succeeds, it's ensured that the Coordinator\ndeployment hasn't been tampered with."}),"\n",(0,o.jsx)(t.h3,{id:"deploy-mysql",children:"Deploy MySQL"}),"\n",(0,o.jsx)(t.p,{children:"Now that the Coordinator has a manifest set, which defines the MySQL deployment as an allowed workload,\nwe can deploy the application:"}),"\n",(0,o.jsx)(t.pre,{children:(0,o.jsx)(t.code,{className:"language-sh",children:"kubectl apply -f deployment/\n"})}),"\n",(0,o.jsx)(t.admonition,{title:"Persistent workload secrets",type:"note",children:(0,o.jsxs)(t.p,{children:["During the initialization process of the workload pod, the Contrast Initializer\nsends an attestation report to the Coordinator and receives a workload secret\nderived from the Coordinator's secret seed and the workload secret ID specified in the\nmanifest, and writes it to a secure in-memory ",(0,o.jsx)(t.code,{children:"volumeMount"}),"."]})}),"\n",(0,o.jsxs)(t.p,{children:["The MySQL deployment is declared as a StatefulSet with a mounted block device.\nAn init container running ",(0,o.jsx)(t.code,{children:"cryptsetup"})," uses the workload secret at\n",(0,o.jsx)(t.code,{children:"/contrast/secrets/workload-secret-seed"})," to generate a key and setup the block\ndevice as a LUKS partition. Before starting the MySQL container, the init\ncontainer uses the generated key to open the LUKS device, which is then mounted\nby the MySQL container. For the MySQL container, this process is completely\ntransparent and works like mounting any other volume. The ",(0,o.jsx)(t.code,{children:"cryptsetup"})," container\nwill remain running to provide the necessary decryption context for the workload\ncontainer."]}),"\n",(0,o.jsx)(t.h2,{id:"verifying-the-deployment-as-a-user",children:"Verifying the deployment as a user"}),"\n",(0,o.jsx)(t.p,{children:"In different scenarios, users of an app may want to verify its security and identity before sharing data, for example, before connecting to the database.\nWith Contrast, a user only needs a single remote-attestation step to verify the deployment - regardless of the size or scale of the deployment.\nContrast is designed such that, by verifying the Coordinator, the user transitively verifies those systems the Coordinator has already verified or will verify in the future.\nSuccessful verification of the Coordinator means that the user can be sure that the given manifest will be enforced."}),"\n",(0,o.jsx)(t.h3,{id:"verifying-the-coordinator",children:"Verifying the Coordinator"}),"\n",(0,o.jsx)(t.p,{children:"A user can verify the Contrast deployment using the verify\ncommand:"}),"\n",(0,o.jsx)(t.pre,{children:(0,o.jsx)(t.code,{className:"language-sh",children:'contrast verify -c "${coordinator}:1313" -m manifest.json\n'})}),"\n",(0,o.jsxs)(t.p,{children:["The CLI will verify the Coordinator via remote attestation using the reference values from a given manifest. This manifest needs\nto be communicated out of band to everyone wanting to verify the deployment, as the ",(0,o.jsx)(t.code,{children:"verify"})," command checks\nif the currently active manifest at the Coordinator matches the manifest given to the CLI. If the command succeeds,\nthe Coordinator deployment was successfully verified to be running in the expected Confidential\nComputing environment with the expected code version. The Coordinator will then return its\nconfiguration over the established TLS channel. The CLI will store this information, namely the root\ncertificate of the mesh (",(0,o.jsx)(t.code,{children:"mesh-ca.pem"}),") and the history of manifests, into the ",(0,o.jsx)(t.code,{children:"verify/"})," directory.\nIn addition, the policies referenced in the manifest history are also written into the same directory."]}),"\n",(0,o.jsx)(t.h3,{id:"auditing-the-manifest-history-and-artifacts",children:"Auditing the manifest history and artifacts"}),"\n",(0,o.jsxs)(t.p,{children:["In the next step, the Coordinator configuration that was written by the ",(0,o.jsx)(t.code,{children:"verify"})," command needs to be audited.\nA user of the application should inspect the manifest and the referenced policies. They could delegate\nthis task to an entity they trust."]}),"\n",(0,o.jsx)(t.h3,{id:"connecting-to-the-application",children:"Connecting to the application"}),"\n",(0,o.jsxs)(t.p,{children:["Other confidential containers can securely connect to the MySQL server via the\n",(0,o.jsx)(t.a,{href:"/contrast/pr-preview/pr-1333/next/components/service-mesh",children:"Service Mesh"}),". The configured ",(0,o.jsx)(t.code,{children:"mysql-client"}),"\ndeployment connects to the MySQL server and inserts test data into a table. To\nview the logs of the ",(0,o.jsx)(t.code,{children:"mysql-client"})," deployment, use the following commands:"]}),"\n",(0,o.jsx)(t.pre,{children:(0,o.jsx)(t.code,{className:"language-sh",children:"kubectl logs -l app.kubernetes.io/name=mysql-client -c mysql-client\n"})}),"\n",(0,o.jsx)(t.p,{children:"The Service Mesh ensures an mTLS connection between the MySQL client and server\nusing the mesh certificates. As a result, no other workload can connect to the\nMySQL server unless explicitly allowed in the manifest."}),"\n",(0,o.jsx)(t.h2,{id:"updating-the-deployment",children:"Updating the deployment"}),"\n",(0,o.jsxs)(t.p,{children:["Because the workload secret is derived from the ",(0,o.jsx)(t.code,{children:"WorkloadSecredID"})," specified in\nthe manifest and not to an individual pod, once the pod restarts, the\n",(0,o.jsx)(t.code,{children:"cryptsetup"})," init container can deterministically generate the same key again\nand open the already partitioned LUKS device.\nFor more information on using the workload secret, see ",(0,o.jsx)(t.a,{href:"/contrast/pr-preview/pr-1333/next/architecture/secrets#workload-secrets",children:"Workload\nSecrets"}),"."]}),"\n",(0,o.jsxs)(t.p,{children:["For example, after making changes to the deployment files, the runtime policies\nneed to be regenerated with ",(0,o.jsx)(t.code,{children:"contrast generate"})," and the new manifest needs to be\nset using ",(0,o.jsx)(t.code,{children:"contrast set"}),"."]}),"\n",(0,o.jsx)(t.pre,{children:(0,o.jsx)(t.code,{className:"language-sh",children:'contrast generate deployment/\ncontrast set -c "${coordinator}:1313" deployment/\n'})}),"\n",(0,o.jsx)(t.p,{children:"The new deployment can then be applied by running:"}),"\n",(0,o.jsx)(t.pre,{children:(0,o.jsx)(t.code,{className:"language-sh",children:"kubectl rollout restart statefulset/mysql-backend\nkubectl rollout restart deployment/mysql-client\n"})}),"\n",(0,o.jsxs)(t.p,{children:["The new MySQL backend pod will then start up the ",(0,o.jsx)(t.code,{children:"cryptsetup"})," init container which\nreceives the same workload secret as before and can therefore generate the\ncorrect key to open the LUKS device. All previously stored data in the MySQL\ndatabase is available in the newly created pod in an encrypted volume mount."]})]})}function h(e={}){const{wrapper:t}={...(0,s.R)(),...e.components};return t?(0,o.jsx)(t,{...e,children:(0,o.jsx)(c,{...e})}):c(e)}function p(e,t){throw new Error("Expected "+(t?"component":"object")+" `"+e+"` to be defined: you likely forgot to import, pass, or provide it.")}},28453:(e,t,n)=>{n.d(t,{R:()=>i,x:()=>a});var r=n(96540);const o={},s=r.createContext(o);function i(e){const t=r.useContext(s);return r.useMemo((function(){return"function"==typeof e?e(t):{...t,...e}}),[t,e])}function a(e){let t;return t=e.disableParentContext?"function"==typeof e.components?e.components(o):e.components||o:i(e.components),r.createElement(s.Provider,{value:t},e.children)}}}]);