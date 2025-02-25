"use strict";(self.webpackChunkcontrast_docs=self.webpackChunkcontrast_docs||[]).push([[2472],{6839:(e,t,n)=>{n.r(t),n.d(t,{assets:()=>l,contentTitle:()=>r,default:()=>h,frontMatter:()=>a,metadata:()=>i,toc:()=>c});const i=JSON.parse('{"id":"examples/emojivoto","title":"Confidential emoji voting","description":"screenshot of the emojivoto UI","source":"@site/docs/examples/emojivoto.md","sourceDirName":"examples","slug":"/examples/emojivoto","permalink":"/contrast/pr-preview/pr-1249/next/examples/emojivoto","draft":false,"unlisted":false,"editUrl":"https://github.com/edgelesssys/contrast/edit/main/docs/docs/examples/emojivoto.md","tags":[],"version":"current","frontMatter":{},"sidebar":"docs","previous":{"title":"Bare metal setup","permalink":"/contrast/pr-preview/pr-1249/next/getting-started/bare-metal"},"next":{"title":"Encrypted volume mount","permalink":"/contrast/pr-preview/pr-1249/next/examples/mysql"}}');var o=n(74848),s=n(28453);const a={},r="Confidential emoji voting",l={},c=[{value:"Prerequisites",id:"prerequisites",level:2},{value:"Steps to deploy emojivoto with Contrast",id:"steps-to-deploy-emojivoto-with-contrast",level:2},{value:"Download the deployment files",id:"download-the-deployment-files",level:3},{value:"Deploy the Contrast runtime",id:"deploy-the-contrast-runtime",level:3},{value:"Deploy the Contrast Coordinator",id:"deploy-the-contrast-coordinator",level:3},{value:"Generate policy annotations and manifest",id:"generate-policy-annotations-and-manifest",level:3},{value:"Set the manifest",id:"set-the-manifest",level:3},{value:"Deploy emojivoto",id:"deploy-emojivoto",level:3},{value:"Verifying the deployment as a user",id:"verifying-the-deployment-as-a-user",level:2},{value:"Verifying the Coordinator",id:"verifying-the-coordinator",level:3},{value:"Auditing the manifest history and artifacts",id:"auditing-the-manifest-history-and-artifacts",level:3},{value:"Connecting securely to the application",id:"connecting-securely-to-the-application",level:3},{value:"Updating the certificate SAN and the manifest (optional)",id:"updating-the-certificate-san-and-the-manifest-optional",level:2},{value:"Configuring the service SAN in the manifest",id:"configuring-the-service-san-in-the-manifest",level:3},{value:"Updating the manifest",id:"updating-the-manifest",level:3},{value:"Rolling out the update",id:"rolling-out-the-update",level:3}];function d(e){const t={a:"a",admonition:"admonition",code:"code",h1:"h1",h2:"h2",h3:"h3",header:"header",img:"img",li:"li",p:"p",pre:"pre",strong:"strong",ul:"ul",...(0,s.R)(),...e.components},{TabItem:i,Tabs:a}=t;return i||p("TabItem",!0),a||p("Tabs",!0),(0,o.jsxs)(o.Fragment,{children:[(0,o.jsx)(t.header,{children:(0,o.jsx)(t.h1,{id:"confidential-emoji-voting",children:"Confidential emoji voting"})}),"\n",(0,o.jsx)(t.p,{children:(0,o.jsx)(t.img,{alt:"screenshot of the emojivoto UI",src:n(72866).A+"",width:"1503",height:"732"})}),"\n",(0,o.jsx)(t.p,{children:(0,o.jsxs)(t.strong,{children:["This tutorial guides you through deploying ",(0,o.jsx)(t.a,{href:"https://github.com/BuoyantIO/emojivoto",children:"emojivoto"})," as a\nconfidential Contrast deployment and validating the deployment from a voter's perspective."]})}),"\n",(0,o.jsxs)(t.p,{children:["Emojivoto is an example app allowing users to vote for different emojis and view votes\non a leader board. It has a microservice architecture consisting of a\nweb frontend (",(0,o.jsx)(t.code,{children:"web"}),"), a gRPC backend for listing available emojis (",(0,o.jsx)(t.code,{children:"emoji"}),"), and a backend for\nthe voting and leader board logic (",(0,o.jsx)(t.code,{children:"voting"}),"). The ",(0,o.jsx)(t.code,{children:"vote-bot"})," simulates user traffic by submitting\nvotes to the frontend."]}),"\n",(0,o.jsx)(t.p,{children:"Emojivoto can be seen as a lighthearted example of an app dealing with sensitive data.\nContrast protects emojivoto in two ways. First, it shields emojivoto as a whole from the infrastructure, for example, Azure.\nSecond, it can be configured to also prevent data access even from the administrator of the app. In the case of emojivoto, this gives assurance to users that their votes remain secret."}),"\n",(0,o.jsx)(t.p,{children:(0,o.jsx)(t.img,{src:"https://raw.githubusercontent.com/BuoyantIO/emojivoto/e490d5789086e75933a474b22f9723fbfa0b29ba/assets/emojivoto-topology.png",alt:"emojivoto components topology"})}),"\n",(0,o.jsx)(t.h2,{id:"prerequisites",children:"Prerequisites"}),"\n",(0,o.jsxs)(t.ul,{children:["\n",(0,o.jsx)(t.li,{children:"Installed Contrast CLI"}),"\n",(0,o.jsxs)(t.li,{children:["A running Kubernetes cluster with support for confidential containers, either on ",(0,o.jsx)(t.a,{href:"/contrast/pr-preview/pr-1249/next/getting-started/cluster-setup",children:"AKS"})," or on ",(0,o.jsx)(t.a,{href:"/contrast/pr-preview/pr-1249/next/getting-started/bare-metal",children:"bare metal"}),"."]}),"\n"]}),"\n",(0,o.jsx)(t.h2,{id:"steps-to-deploy-emojivoto-with-contrast",children:"Steps to deploy emojivoto with Contrast"}),"\n",(0,o.jsx)(t.h3,{id:"download-the-deployment-files",children:"Download the deployment files"}),"\n",(0,o.jsx)(t.p,{children:"The emojivoto deployment files are part of the Contrast release. You can download them by running:"}),"\n",(0,o.jsx)(t.pre,{children:(0,o.jsx)(t.code,{className:"language-sh",children:"curl -fLO https://github.com/edgelesssys/contrast/releases/latest/download/emojivoto-demo.yml --create-dirs --output-dir deployment\n"})}),"\n",(0,o.jsx)(t.h3,{id:"deploy-the-contrast-runtime",children:"Deploy the Contrast runtime"}),"\n",(0,o.jsxs)(t.p,{children:["Contrast depends on a ",(0,o.jsx)(t.a,{href:"/contrast/pr-preview/pr-1249/next/components/runtime",children:"custom Kubernetes RuntimeClass"}),",\nwhich needs to be installed to the cluster initially.\nThis consists of a ",(0,o.jsx)(t.code,{children:"RuntimeClass"})," resource and a ",(0,o.jsx)(t.code,{children:"DaemonSet"})," that performs installation on worker nodes.\nThis step is only required once for each version of the runtime.\nIt can be shared between Contrast deployments."]}),"\n",(0,o.jsxs)(a,{queryString:"platform",children:[(0,o.jsx)(i,{value:"aks-clh-snp",label:"AKS",default:!0,children:(0,o.jsx)(t.pre,{children:(0,o.jsx)(t.code,{className:"language-sh",children:"kubectl apply -f https://github.com/edgelesssys/contrast/releases/latest/download/runtime-aks-clh-snp.yml\n"})})}),(0,o.jsx)(i,{value:"k3s-qemu-snp",label:"Bare metal (SEV-SNP)",children:(0,o.jsx)(t.pre,{children:(0,o.jsx)(t.code,{className:"language-sh",children:"kubectl apply -f https://github.com/edgelesssys/contrast/releases/latest/download/runtime-k3s-qemu-snp.yml\n"})})}),(0,o.jsx)(i,{value:"k3s-qemu-tdx",label:"Bare metal (TDX)",children:(0,o.jsx)(t.pre,{children:(0,o.jsx)(t.code,{className:"language-sh",children:"kubectl apply -f https://github.com/edgelesssys/contrast/releases/latest/download/runtime-k3s-qemu-tdx.yml\n"})})})]}),"\n",(0,o.jsx)(t.h3,{id:"deploy-the-contrast-coordinator",children:"Deploy the Contrast Coordinator"}),"\n",(0,o.jsx)(t.p,{children:"Deploy the Contrast Coordinator, comprising a single replica deployment and a\nLoadBalancer service, into your cluster:"}),"\n",(0,o.jsxs)(a,{queryString:"platform",children:[(0,o.jsx)(i,{value:"aks-clh-snp",label:"AKS",default:!0,children:(0,o.jsx)(t.pre,{children:(0,o.jsx)(t.code,{className:"language-sh",children:"kubectl apply -f https://github.com/edgelesssys/contrast/releases/latest/download/coordinator-aks-clh-snp.yml\n"})})}),(0,o.jsx)(i,{value:"k3s-qemu-snp",label:"Bare metal (SEV-SNP)",children:(0,o.jsx)(t.pre,{children:(0,o.jsx)(t.code,{className:"language-sh",children:"kubectl apply -f https://github.com/edgelesssys/contrast/releases/latest/download/coordinator-k3s-qemu-snp.yml\n"})})}),(0,o.jsx)(i,{value:"k3s-qemu-tdx",label:"Bare metal (TDX)",children:(0,o.jsx)(t.pre,{children:(0,o.jsx)(t.code,{className:"language-sh",children:"kubectl apply -f https://github.com/edgelesssys/contrast/releases/latest/download/coordinator-k3s-qemu-tdx.yml\n"})})})]}),"\n",(0,o.jsx)(t.h3,{id:"generate-policy-annotations-and-manifest",children:"Generate policy annotations and manifest"}),"\n",(0,o.jsxs)(t.p,{children:["Run the ",(0,o.jsx)(t.code,{children:"generate"})," command to generate the execution policies and add them as\nannotations to your deployment files. A ",(0,o.jsx)(t.code,{children:"manifest.json"})," file with the reference values\nof your deployment will be created:"]}),"\n",(0,o.jsxs)(a,{queryString:"platform",children:[(0,o.jsx)(i,{value:"aks-clh-snp",label:"AKS",default:!0,children:(0,o.jsx)(t.pre,{children:(0,o.jsx)(t.code,{className:"language-sh",children:"contrast generate --reference-values aks-clh-snp deployment/\n"})})}),(0,o.jsxs)(i,{value:"k3s-qemu-snp",label:"Bare metal (SEV-SNP)",children:[(0,o.jsx)(t.pre,{children:(0,o.jsx)(t.code,{className:"language-sh",children:"contrast generate --reference-values k3s-qemu-snp deployment/\n"})}),(0,o.jsx)(t.admonition,{title:"Missing TCB values",type:"note",children:(0,o.jsxs)(t.p,{children:["On bare-metal SEV-SNP, ",(0,o.jsx)(t.code,{children:"contrast generate"})," is unable to fill in the ",(0,o.jsx)(t.code,{children:"MinimumTCB"})," values as they can vary between platforms.\nThey will have to be filled in manually.\nIf you don't know the correct values use ",(0,o.jsx)(t.code,{children:'{"BootloaderVersion":255,"TEEVersion":255,"SNPVersion":255,"MicrocodeVersion":255}'})," and observe the real values in the error messages in the following steps. This should only be done in a secure environment. Note that the values will differ between CPU models."]})})]}),(0,o.jsxs)(i,{value:"k3s-qemu-tdx",label:"Bare metal (TDX)",children:[(0,o.jsx)(t.pre,{children:(0,o.jsx)(t.code,{className:"language-sh",children:"contrast generate --reference-values k3s-qemu-tdx deployment/\n"})}),(0,o.jsx)(t.admonition,{title:"Missing TCB values",type:"note",children:(0,o.jsxs)(t.p,{children:["On bare-metal TDX, ",(0,o.jsx)(t.code,{children:"contrast generate"})," is unable to fill in the ",(0,o.jsx)(t.code,{children:"MinimumTeeTcbSvn"})," and ",(0,o.jsx)(t.code,{children:"MrSeam"})," TCB values as they can vary between platforms.\nThey will have to be filled in manually.\nIf you don't know the correct values use ",(0,o.jsx)(t.code,{children:"ffffffffffffffffffffffffffffffff"})," and ",(0,o.jsx)(t.code,{children:"000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"})," respectively and observe the real values in the error messages in the following steps. This should only be done in a secure environment."]})})]})]}),"\n",(0,o.jsxs)(t.admonition,{title:"Runtime class and Initializer",type:"note",children:[(0,o.jsxs)(t.p,{children:["The deployment YAML shipped for this demo is already configured to be used with Contrast.\nA ",(0,o.jsx)(t.a,{href:"../components/runtime",children:"runtime class"})," ",(0,o.jsx)(t.code,{children:"contrast-cc"}),"\nwas added to the pods to signal they should be run as Confidential Containers. During the generation process,\nthe Contrast ",(0,o.jsx)(t.a,{href:"/contrast/pr-preview/pr-1249/next/components/overview#the-initializer",children:"Initializer"})," will be added as an init container to these\nworkloads to facilitate the attestation and certificate pulling before the actual workload is started."]}),(0,o.jsxs)(t.p,{children:["Further, the deployment YAML is also configured with the Contrast ",(0,o.jsx)(t.a,{href:"/contrast/pr-preview/pr-1249/next/components/service-mesh",children:"service mesh"}),".\nThe configured service mesh proxy provides transparent protection for the communication between\nthe different components of emojivoto."]})]}),"\n",(0,o.jsx)(t.h3,{id:"set-the-manifest",children:"Set the manifest"}),"\n",(0,o.jsx)(t.p,{children:"Configure the coordinator with a manifest. It might take up to a few minutes\nfor the load balancer to be created and the Coordinator being available."}),"\n",(0,o.jsx)(t.pre,{children:(0,o.jsx)(t.code,{className:"language-sh",children:'coordinator=$(kubectl get svc coordinator -o=jsonpath=\'{.status.loadBalancer.ingress[0].ip}\')\necho "The user API of your Contrast Coordinator is available at $coordinator:1313"\ncontrast set -c "${coordinator}:1313" deployment/\n'})}),"\n",(0,o.jsx)(t.p,{children:"The CLI will use the reference values from the manifest to attest the Coordinator deployment\nduring the TLS handshake. If the connection succeeds, it's ensured that the Coordinator\ndeployment hasn't been tampered with."}),"\n",(0,o.jsx)(t.admonition,{type:"warning",children:(0,o.jsxs)(t.p,{children:["On bare metal, the ",(0,o.jsx)(t.a,{href:"/contrast/pr-preview/pr-1249/next/components/policies#platform-differences",children:"coordinator policy hash"})," must be overwritten using ",(0,o.jsx)(t.code,{children:"--coordinator-policy-hash"}),"."]})}),"\n",(0,o.jsx)(t.h3,{id:"deploy-emojivoto",children:"Deploy emojivoto"}),"\n",(0,o.jsx)(t.p,{children:"Now that the coordinator has a manifest set, which defines the emojivoto deployment as an allowed workload,\nwe can deploy the application:"}),"\n",(0,o.jsx)(t.pre,{children:(0,o.jsx)(t.code,{className:"language-sh",children:"kubectl apply -f deployment/\n"})}),"\n",(0,o.jsx)(t.admonition,{title:"Inter-deployment communication",type:"note",children:(0,o.jsxs)(t.p,{children:["The Contrast Coordinator issues mesh certificates after successfully validating workloads.\nThese certificates can be used for secure inter-deployment communication. The Initializer\nsends an attestation report to the Coordinator, retrieves certificates and a private key in return\nand writes them to a ",(0,o.jsx)(t.code,{children:"volumeMount"}),". The service mesh sidecar is configured to use the credentials\nfrom the ",(0,o.jsx)(t.code,{children:"volumeMount"})," when communicating with other parts of the deployment over mTLS.\nThe public facing frontend for voting uses the mesh certificate without client authentication."]})}),"\n",(0,o.jsx)(t.h2,{id:"verifying-the-deployment-as-a-user",children:"Verifying the deployment as a user"}),"\n",(0,o.jsx)(t.p,{children:"In different scenarios, users of an app may want to verify its security and identity before sharing data, for example, before casting a vote.\nWith Contrast, a user only needs a single remote-attestation step to verify the deployment - regardless of the size or scale of the deployment.\nContrast is designed such that, by verifying the Coordinator, the user transitively verifies those systems the Coordinator has already verified or will verify in the future.\nSuccessful verification of the Coordinator means that the user can be sure that the given manifest will be enforced."}),"\n",(0,o.jsx)(t.h3,{id:"verifying-the-coordinator",children:"Verifying the Coordinator"}),"\n",(0,o.jsx)(t.p,{children:"A user can verify the Contrast deployment using the verify\ncommand:"}),"\n",(0,o.jsx)(t.pre,{children:(0,o.jsx)(t.code,{className:"language-sh",children:'contrast verify -c "${coordinator}:1313" -m manifest.json\n'})}),"\n",(0,o.jsxs)(t.p,{children:["The CLI will verify the Coordinator via remote attestation using the reference values from a given manifest. This manifest needs\nto be communicated out of band to everyone wanting to verify the deployment, as the ",(0,o.jsx)(t.code,{children:"verify"})," command checks\nif the currently active manifest at the Coordinator matches the manifest given to the CLI. If the command succeeds,\nthe Coordinator deployment was successfully verified to be running in the expected Confidential\nComputing environment with the expected code version. The Coordinator will then return its\nconfiguration over the established TLS channel. The CLI will store this information, namely the root\ncertificate of the mesh (",(0,o.jsx)(t.code,{children:"mesh-ca.pem"}),") and the history of manifests, into the ",(0,o.jsx)(t.code,{children:"verify/"})," directory.\nIn addition, the policies referenced in the manifest history are also written into the same directory."]}),"\n",(0,o.jsx)(t.admonition,{type:"warning",children:(0,o.jsxs)(t.p,{children:["On bare metal, the ",(0,o.jsx)(t.a,{href:"/contrast/pr-preview/pr-1249/next/components/policies#platform-differences",children:"coordinator policy hash"})," must be overwritten using ",(0,o.jsx)(t.code,{children:"--coordinator-policy-hash"}),"."]})}),"\n",(0,o.jsx)(t.h3,{id:"auditing-the-manifest-history-and-artifacts",children:"Auditing the manifest history and artifacts"}),"\n",(0,o.jsxs)(t.p,{children:["In the next step, the Coordinator configuration that was written by the ",(0,o.jsx)(t.code,{children:"verify"})," command needs to be audited.\nA potential voter should inspect the manifest and the referenced policies. They could delegate\nthis task to an entity they trust."]}),"\n",(0,o.jsx)(t.h3,{id:"connecting-securely-to-the-application",children:"Connecting securely to the application"}),"\n",(0,o.jsxs)(t.p,{children:["After ensuring the configuration of the Coordinator fits the expectation, the user can securely connect\nto the application using the Coordinator's ",(0,o.jsx)(t.code,{children:"mesh-ca.pem"})," as a trusted CA certificate."]}),"\n",(0,o.jsx)(t.p,{children:"To access the web frontend, expose the service on a public IP address via a LoadBalancer service:"}),"\n",(0,o.jsx)(t.pre,{children:(0,o.jsx)(t.code,{className:"language-sh",children:"frontendIP=$(kubectl get svc web-svc -o=jsonpath='{.status.loadBalancer.ingress[0].ip}')\necho \"Frontend is available at  https://$frontendIP, you can visit it in your browser.\"\n"})}),"\n",(0,o.jsxs)(t.p,{children:["Using ",(0,o.jsx)(t.code,{children:"openssl"}),", the certificate of the service can be validated with the ",(0,o.jsx)(t.code,{children:"mesh-ca.pem"}),":"]}),"\n",(0,o.jsx)(t.pre,{children:(0,o.jsx)(t.code,{className:"language-sh",children:"openssl s_client -CAfile verify/mesh-ca.pem -verify_return_error -connect ${frontendIP}:443 < /dev/null\n"})}),"\n",(0,o.jsx)(t.h2,{id:"updating-the-certificate-san-and-the-manifest-optional",children:"Updating the certificate SAN and the manifest (optional)"}),"\n",(0,o.jsx)(t.p,{children:"By default, mesh certificates are issued with a wildcard DNS entry. The web frontend is accessed\nvia load balancer IP in this demo. Tools like curl check the certificate for IP entries in the subject alternative name (SAN) field.\nValidation fails since the certificate contains no IP entries as a SAN.\nFor example, a connection attempt using the curl and the mesh CA certificate with throw the following error:"}),"\n",(0,o.jsx)(t.pre,{children:(0,o.jsx)(t.code,{className:"language-sh",children:"$ curl --cacert ./verify/mesh-ca.pem \"https://${frontendIP}:443\"\ncurl: (60) SSL: no alternative certificate subject name matches target host name '203.0.113.34'\n"})}),"\n",(0,o.jsx)(t.h3,{id:"configuring-the-service-san-in-the-manifest",children:"Configuring the service SAN in the manifest"}),"\n",(0,o.jsxs)(t.p,{children:["The ",(0,o.jsx)(t.code,{children:"Policies"})," section of the manifest maps policy hashes to a list of SANs. To enable certificate verification\nof the web frontend with tools like curl, edit the policy with your favorite editor and add the ",(0,o.jsx)(t.code,{children:"frontendIP"})," to\nthe list that already contains the ",(0,o.jsx)(t.code,{children:'"web"'})," DNS entry:"]}),"\n",(0,o.jsx)(t.pre,{children:(0,o.jsx)(t.code,{className:"language-diff",children:'   "Policies": {\n     ...\n     "99dd77cbd7fe2c4e1f29511014c14054a21a376f7d58a48d50e9e036f4522f6b": {\n       "SANs": [\n         "web",\n-        "*"\n+        "*",\n+        "203.0.113.34"\n       ],\n       "WorkloadSecretID": "web"\n      },\n'})}),"\n",(0,o.jsx)(t.h3,{id:"updating-the-manifest",children:"Updating the manifest"}),"\n",(0,o.jsx)(t.p,{children:"Next, set the changed manifest at the coordinator with:"}),"\n",(0,o.jsx)(t.pre,{children:(0,o.jsx)(t.code,{className:"language-sh",children:'contrast set -c "${coordinator}:1313" deployment/\n'})}),"\n",(0,o.jsxs)(t.p,{children:["The Contrast Coordinator will rotate the mesh ca certificate on the manifest update. Workload certificates issued\nafter the manifest update are thus issued by another certificate authority and services receiving the new CA certificate chain\nwon't trust parts of the deployment that got their certificate issued before the update. This way, Contrast ensures\nthat parts of the deployment that received a security update won't be infected by parts of the deployment at an older\npatch level that may have been compromised. The ",(0,o.jsx)(t.code,{children:"mesh-ca.pem"})," is updated with the new CA certificate chain."]}),"\n",(0,o.jsx)(t.admonition,{type:"warning",children:(0,o.jsxs)(t.p,{children:["On bare metal, the ",(0,o.jsx)(t.a,{href:"/contrast/pr-preview/pr-1249/next/components/policies#platform-differences",children:"coordinator policy hash"})," must be overwritten using ",(0,o.jsx)(t.code,{children:"--coordinator-policy-hash"}),"."]})}),"\n",(0,o.jsx)(t.h3,{id:"rolling-out-the-update",children:"Rolling out the update"}),"\n",(0,o.jsx)(t.p,{children:"The Coordinator has the new manifest set, but the different containers of the app are still\nusing the older certificate authority. The Contrast Initializer terminates after the initial attestation\nflow and won't pull new certificates on manifest updates."}),"\n",(0,o.jsx)(t.p,{children:"To roll out the update, use:"}),"\n",(0,o.jsx)(t.pre,{children:(0,o.jsx)(t.code,{className:"language-sh",children:"kubectl rollout restart deployment/emoji\nkubectl rollout restart deployment/vote-bot\nkubectl rollout restart deployment/voting\nkubectl rollout restart deployment/web\n"})}),"\n",(0,o.jsx)(t.p,{children:"After the update has been rolled out, connecting to the frontend using curl will successfully validate\nthe service certificate and return the HTML document of the voting site:"}),"\n",(0,o.jsx)(t.pre,{children:(0,o.jsx)(t.code,{className:"language-sh",children:'curl --cacert ./mesh-ca.pem "https://${frontendIP}:443"\n'})})]})}function h(e={}){const{wrapper:t}={...(0,s.R)(),...e.components};return t?(0,o.jsx)(t,{...e,children:(0,o.jsx)(d,{...e})}):d(e)}function p(e,t){throw new Error("Expected "+(t?"component":"object")+" `"+e+"` to be defined: you likely forgot to import, pass, or provide it.")}},72866:(e,t,n)=>{n.d(t,{A:()=>i});const i=n.p+"assets/images/emoijvoto-3fb48da575d6d4adf76cc90caa35d762.png"},28453:(e,t,n)=>{n.d(t,{R:()=>a,x:()=>r});var i=n(96540);const o={},s=i.createContext(o);function a(e){const t=i.useContext(s);return i.useMemo((function(){return"function"==typeof e?e(t):{...t,...e}}),[t,e])}function r(e){let t;return t=e.disableParentContext?"function"==typeof e.components?e.components(o):e.components||o:a(e.components),i.createElement(s.Provider,{value:t},e.children)}}}]);