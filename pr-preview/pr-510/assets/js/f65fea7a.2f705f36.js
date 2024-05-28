"use strict";(self.webpackChunkcontrast_docs=self.webpackChunkcontrast_docs||[]).push([[2472],{7970:(e,t,n)=>{n.r(t),n.d(t,{assets:()=>d,contentTitle:()=>a,default:()=>h,frontMatter:()=>s,metadata:()=>r,toc:()=>l});var o=n(4848),i=n(8453);const s={},a="Confidential emoji voting",r={id:"examples/emojivoto",title:"Confidential emoji voting",description:"screenshot of the emojivoto UI",source:"@site/docs/examples/emojivoto.md",sourceDirName:"examples",slug:"/examples/emojivoto",permalink:"/contrast/pr-preview/pr-510/next/examples/emojivoto",draft:!1,unlisted:!1,editUrl:"https://github.com/edgelesssys/contrast/edit/main/docs/docs/examples/emojivoto.md",tags:[],version:"current",frontMatter:{},sidebar:"docs",previous:{title:"Examples",permalink:"/contrast/pr-preview/pr-510/next/examples/"},next:{title:"Workload deployment",permalink:"/contrast/pr-preview/pr-510/next/deployment"}},d={},l=[{value:"Motivation",id:"motivation",level:3},{value:"Prerequisites",id:"prerequisites",level:2},{value:"Steps to deploy emojivoto with Contrast",id:"steps-to-deploy-emojivoto-with-contrast",level:2},{value:"Downloading the deployment",id:"downloading-the-deployment",level:3},{value:"Deploy the Contrast runtime",id:"deploy-the-contrast-runtime",level:3},{value:"Deploy the Contrast Coordinator",id:"deploy-the-contrast-coordinator",level:3},{value:"Generate policy annotations and manifest",id:"generate-policy-annotations-and-manifest",level:3},{value:"Set the manifest",id:"set-the-manifest",level:3},{value:"Deploy emojivoto",id:"deploy-emojivoto",level:3},{value:"Voter&#39;s perspective: Verifying the ballot",id:"voters-perspective-verifying-the-ballot",level:2},{value:"Attest the Coordinator",id:"attest-the-coordinator",level:3},{value:"Manifest history and artifact audit",id:"manifest-history-and-artifact-audit",level:3},{value:"Confidential connection to the attested workload",id:"confidential-connection-to-the-attested-workload",level:3},{value:"Certificate SAN and manifest update (optional)",id:"certificate-san-and-manifest-update-optional",level:2},{value:"Configure the service SAN in the manifest",id:"configure-the-service-san-in-the-manifest",level:3},{value:"Update the manifest",id:"update-the-manifest",level:3},{value:"Rolling out the update",id:"rolling-out-the-update",level:3}];function c(e){const t={a:"a",admonition:"admonition",code:"code",h1:"h1",h2:"h2",h3:"h3",img:"img",li:"li",p:"p",pre:"pre",strong:"strong",ul:"ul",...(0,i.R)(),...e.components};return(0,o.jsxs)(o.Fragment,{children:[(0,o.jsx)(t.h1,{id:"confidential-emoji-voting",children:"Confidential emoji voting"}),"\n",(0,o.jsx)(t.p,{children:(0,o.jsx)(t.img,{alt:"screenshot of the emojivoto UI",src:n(2866).A+"",width:"1503",height:"732"})}),"\n",(0,o.jsx)(t.p,{children:(0,o.jsxs)(t.strong,{children:["This tutorial guides you through deploying ",(0,o.jsx)(t.a,{href:"https://github.com/BuoyantIO/emojivoto",children:"emojivoto"})," as a\nconfidential Contrast deployment and validating the deployment from a voters perspective."]})}),"\n",(0,o.jsxs)(t.p,{children:["Emojivoto is an example app allowing users to vote for different emojis and view votes\non a leader board. It has a microservice architecture consisting of a\nweb frontend (",(0,o.jsx)(t.code,{children:"web"}),"), a gRPC backend for listing available emojis (",(0,o.jsx)(t.code,{children:"emoji"}),"), and a backend for\nthe voting and leader board logic (",(0,o.jsx)(t.code,{children:"voting"}),"). The ",(0,o.jsx)(t.code,{children:"vote-bot"})," simulates user traffic by submitting\nvotes to the frontend."]}),"\n",(0,o.jsx)(t.p,{children:(0,o.jsx)(t.img,{src:"https://raw.githubusercontent.com/BuoyantIO/emojivoto/e490d5789086e75933a474b22f9723fbfa0b29ba/assets/emojivoto-topology.png",alt:"emojivoto components topology"})}),"\n",(0,o.jsx)(t.h3,{id:"motivation",children:"Motivation"}),"\n",(0,o.jsx)(t.p,{children:"Using a voting service, users' votes are considered highly sensitive data, as we require\na secret ballot. Also, users are likely interested in the fairness of the ballot. For\nboth requirements, we can use Confidential Computing and, specifically, workload attestation\nto prove to those interested in voting that the app is running in a protected environment\nwhere their votes are processed without leaking to the platform provider or workload owner."}),"\n",(0,o.jsx)(t.h2,{id:"prerequisites",children:"Prerequisites"}),"\n",(0,o.jsxs)(t.ul,{children:["\n",(0,o.jsxs)(t.li,{children:[(0,o.jsx)(t.strong,{children:"Installed Contrast CLI."}),"\nSee the ",(0,o.jsx)(t.a,{href:"/contrast/pr-preview/pr-510/next/getting-started/install",children:"installation instructions"})," on how to get it."]}),"\n",(0,o.jsxs)(t.li,{children:[(0,o.jsx)(t.strong,{children:"Running cluster with Confidential Containers support."}),"\nPlease follow the ",(0,o.jsx)(t.a,{href:"/contrast/pr-preview/pr-510/next/getting-started/cluster-setup",children:"cluster setup instructions"}),"\nto create a cluster."]}),"\n"]}),"\n",(0,o.jsx)(t.h2,{id:"steps-to-deploy-emojivoto-with-contrast",children:"Steps to deploy emojivoto with Contrast"}),"\n",(0,o.jsx)(t.h3,{id:"downloading-the-deployment",children:"Downloading the deployment"}),"\n",(0,o.jsx)(t.p,{children:"The emojivoto deployment files are part of a zip file in the Contrast release. You can download the\nlatest deployment by running:"}),"\n",(0,o.jsx)(t.pre,{children:(0,o.jsx)(t.code,{className:"language-sh",children:"curl -fLO https://github.com/edgelesssys/contrast/releases/latest/download/emojivoto-demo.zip\n"})}),"\n",(0,o.jsxs)(t.p,{children:["After that, unzip the ",(0,o.jsx)(t.code,{children:"emojivoto-demo.zip"})," file to extract the ",(0,o.jsx)(t.code,{children:"deployment/"})," directory."]}),"\n",(0,o.jsx)(t.pre,{children:(0,o.jsx)(t.code,{className:"language-sh",children:"unzip emojivoto-demo.zip\n"})}),"\n",(0,o.jsx)(t.h3,{id:"deploy-the-contrast-runtime",children:"Deploy the Contrast runtime"}),"\n",(0,o.jsxs)(t.p,{children:["Contrast depends on a ",(0,o.jsxs)(t.a,{href:"/contrast/pr-preview/pr-510/next/components/runtime",children:["custom Kubernetes ",(0,o.jsx)(t.code,{children:"RuntimeClass"})," (",(0,o.jsx)(t.code,{children:"contrast-cc"}),")"]}),",\nwhich needs to be installed in the cluster prior to the Coordinator or any confidential workloads.\nThis consists of a ",(0,o.jsx)(t.code,{children:"RuntimeClass"})," resource and a ",(0,o.jsx)(t.code,{children:"DaemonSet"})," that performs installation on worker nodes.\nThis step is only required once for each version of the runtime.\nIt can be shared between Contrast deployments."]}),"\n",(0,o.jsx)(t.pre,{children:(0,o.jsx)(t.code,{className:"language-sh",children:"kubectl apply -f https://github.com/edgelesssys/contrast/releases/latest/download/runtime.yml\n"})}),"\n",(0,o.jsx)(t.h3,{id:"deploy-the-contrast-coordinator",children:"Deploy the Contrast Coordinator"}),"\n",(0,o.jsx)(t.p,{children:"Deploy the Contrast Coordinator, comprising a single replica deployment and a\nLoadBalancer service, into your cluster:"}),"\n",(0,o.jsx)(t.pre,{children:(0,o.jsx)(t.code,{className:"language-sh",children:"kubectl apply -f https://github.com/edgelesssys/contrast/releases/latest/download/coordinator.yml\n"})}),"\n",(0,o.jsx)(t.h3,{id:"generate-policy-annotations-and-manifest",children:"Generate policy annotations and manifest"}),"\n",(0,o.jsxs)(t.p,{children:["Run the ",(0,o.jsx)(t.code,{children:"generate"})," command to generate the execution policies and add them as\nannotations to your deployment files. A ",(0,o.jsx)(t.code,{children:"manifest.json"})," file with the reference values\nof your deployment will be created:"]}),"\n",(0,o.jsx)(t.pre,{children:(0,o.jsx)(t.code,{className:"language-sh",children:"contrast generate deployment/\n"})}),"\n",(0,o.jsxs)(t.admonition,{title:"Runtime class and Initializer",type:"note",children:[(0,o.jsxs)(t.p,{children:["The deployment YAML shipped for this demo is already configured to be used with Contrast.\nA ",(0,o.jsx)(t.a,{href:"https://docs.edgeless.systems/contrast/components/runtime",children:"runtime class"})," ",(0,o.jsx)(t.code,{children:"contrast-cc-<VERSIONHASH>"}),"\nwas added to the pods to signal they should be run as Confidential Containers. In addition, the Contrast\n",(0,o.jsx)(t.a,{href:"/contrast/pr-preview/pr-510/next/components/#the-initializer",children:"Initializer"})," was added as an init container to these workloads to\nfacilitate the attestation and certificate pulling before the actual workload is started."]}),(0,o.jsxs)(t.p,{children:["Further, the deployment YAML is also configured with the Contrast ",(0,o.jsx)(t.a,{href:"/contrast/pr-preview/pr-510/next/components/service-mesh",children:"service mesh"}),".\nThe configured service mesh proxy provides transparent protection for the communication between\nthe different components of emojivoto."]})]}),"\n",(0,o.jsx)(t.h3,{id:"set-the-manifest",children:"Set the manifest"}),"\n",(0,o.jsx)(t.p,{children:"Configure the coordinator with a manifest. It might take up to a few minutes\nfor the load balancer to be created and the Coordinator being available."}),"\n",(0,o.jsx)(t.pre,{children:(0,o.jsx)(t.code,{className:"language-sh",children:'coordinator=$(kubectl get svc coordinator -o=jsonpath=\'{.status.loadBalancer.ingress[0].ip}\')\necho "The user API of your Contrast Coordinator is available at $coordinator:1313"\ncontrast set -c "${coordinator}:1313" deployment/\n'})}),"\n",(0,o.jsx)(t.p,{children:"The CLI will use the embedded reference values to attest the Coordinator deployment\nduring the TLS handshake. If the connection succeeds, we're ensured that the Coordinator\ndeployment hasn't been tampered with."}),"\n",(0,o.jsx)(t.h3,{id:"deploy-emojivoto",children:"Deploy emojivoto"}),"\n",(0,o.jsx)(t.p,{children:"Now that the coordinator has a manifest set, which defines the emojivoto deployment as an allowed workload,\nwe can deploy the application:"}),"\n",(0,o.jsx)(t.pre,{children:(0,o.jsx)(t.code,{className:"language-sh",children:"kubectl apply -f deployment/\n"})}),"\n",(0,o.jsx)(t.admonition,{title:"Inter-deployment communication",type:"note",children:(0,o.jsxs)(t.p,{children:["The Contrast Coordinator issues mesh certificates after successfully validating workloads.\nThese certificates can be used for secure inter-deployment communication. The Initializer\nsends an attestation report to the Coordinator, retrieves certificates and a private key in return\nand writes them to a ",(0,o.jsx)(t.code,{children:"volumeMount"}),". The service mesh sidecar is configured to use the credentials\nfrom the ",(0,o.jsx)(t.code,{children:"volumeMount"})," when communicating with other parts of the deployment over mTLS.\nThe public facing frontend for voting uses the mesh certificate without client authentication."]})}),"\n",(0,o.jsx)(t.h2,{id:"voters-perspective-verifying-the-ballot",children:"Voter's perspective: Verifying the ballot"}),"\n",(0,o.jsx)(t.p,{children:"As voters, we want to verify the fairness and confidentiality of the deployment before\ndeciding to vote. Regardless of the scale of our distributed deployment, Contrast only\nneeds a single remote attestation step to verify the deployment. By doing remote attestation\nof the Coordinator, we transitively verify those systems the Coordinator has already attested\nor will attest in the future. Successful verification of the Coordinator means that\nwe can be sure it will enforce the configured manifest."}),"\n",(0,o.jsx)(t.h3,{id:"attest-the-coordinator",children:"Attest the Coordinator"}),"\n",(0,o.jsx)(t.p,{children:"A potential voter can verify the Contrast deployment using the verify\ncommand:"}),"\n",(0,o.jsx)(t.pre,{children:(0,o.jsx)(t.code,{className:"language-sh",children:'contrast verify -c "${coordinator}:1313"\n'})}),"\n",(0,o.jsxs)(t.p,{children:["The CLI will attest the Coordinator using embedded reference values. If the command succeeds,\nthe Coordinator deployment was successfully verified to be running in the expected Confidential\nComputing environment with the expected code version. The Coordinator will then return its\nconfiguration over the established TLS channel. The CLI will store this information, namely the root\ncertificate of the mesh (",(0,o.jsx)(t.code,{children:"mesh-ca.pem"}),") and the history of manifests, into the ",(0,o.jsx)(t.code,{children:"verify/"})," directory.\nIn addition, the policies referenced in the manifest history are also written into the same directory."]}),"\n",(0,o.jsx)(t.h3,{id:"manifest-history-and-artifact-audit",children:"Manifest history and artifact audit"}),"\n",(0,o.jsxs)(t.p,{children:["In the next step, the Coordinator configuration that was written by the ",(0,o.jsx)(t.code,{children:"verify"})," command needs to be audited.\nA potential voter should inspect the manifest and the referenced policies. They could delegate\nthis task to an entity they trust."]}),"\n",(0,o.jsx)(t.h3,{id:"confidential-connection-to-the-attested-workload",children:"Confidential connection to the attested workload"}),"\n",(0,o.jsxs)(t.p,{children:["After ensuring the configuration of the Coordinator fits the expectation, you can securely connect\nto the workloads using the Coordinator's ",(0,o.jsx)(t.code,{children:"mesh-ca.pem"})," as a trusted CA certificate."]}),"\n",(0,o.jsx)(t.p,{children:"To access the web frontend, expose the service on a public IP address via a LoadBalancer service:"}),"\n",(0,o.jsx)(t.pre,{children:(0,o.jsx)(t.code,{className:"language-sh",children:"frontendIP=$(kubectl get svc web-svc -o=jsonpath='{.status.loadBalancer.ingress[0].ip}')\necho \"Frontend is available at  https://$frontendIP, you can visit it in your browser.\"\n"})}),"\n",(0,o.jsxs)(t.p,{children:["Using ",(0,o.jsx)(t.code,{children:"openssl"}),", the certificate of the service can be validated with the ",(0,o.jsx)(t.code,{children:"mesh-ca.pem"}),":"]}),"\n",(0,o.jsx)(t.pre,{children:(0,o.jsx)(t.code,{className:"language-sh",children:"openssl s_client -CAfile verify/mesh-ca.pem -verify_return_error -connect ${frontendIP}:443 < /dev/null\n"})}),"\n",(0,o.jsx)(t.h2,{id:"certificate-san-and-manifest-update-optional",children:"Certificate SAN and manifest update (optional)"}),"\n",(0,o.jsx)(t.p,{children:"By default, mesh certificates are issued with a wildcard DNS entry. The web frontend is accessed\nvia load balancer IP in this demo. Tools like curl check the certificate for IP entries in the SAN field.\nValidation fails since the certificate contains no IP entries as a subject alternative name (SAN).\nFor example, a connection attempt using the curl and the mesh CA certificate with throw the following error:"}),"\n",(0,o.jsx)(t.pre,{children:(0,o.jsx)(t.code,{className:"language-sh",children:"$ curl --cacert ./verify/mesh-ca.pem \"https://${frontendIP}:443\"\ncurl: (60) SSL: no alternative certificate subject name matches target host name '203.0.113.34'\n"})}),"\n",(0,o.jsx)(t.h3,{id:"configure-the-service-san-in-the-manifest",children:"Configure the service SAN in the manifest"}),"\n",(0,o.jsxs)(t.p,{children:["The ",(0,o.jsx)(t.code,{children:"Policies"})," section of the manifest maps policy hashes to a list of SANs. To enable certificate verification\nof the web frontend with tools like curl, edit the policy with your favorite editor and add the ",(0,o.jsx)(t.code,{children:"frontendIP"})," to\nthe list that already contains the ",(0,o.jsx)(t.code,{children:'"web"'})," DNS entry:"]}),"\n",(0,o.jsx)(t.pre,{children:(0,o.jsx)(t.code,{className:"language-diff",children:'   "Policies": {\n     ...\n     "99dd77cbd7fe2c4e1f29511014c14054a21a376f7d58a48d50e9e036f4522f6b": [\n       "web",\n-      "*"\n+      "*",\n+      "203.0.113.34"\n     ],\n'})}),"\n",(0,o.jsx)(t.h3,{id:"update-the-manifest",children:"Update the manifest"}),"\n",(0,o.jsx)(t.p,{children:"Next, set the changed manifest at the coordinator with:"}),"\n",(0,o.jsx)(t.pre,{children:(0,o.jsx)(t.code,{className:"language-sh",children:'contrast set -c "${coordinator}:1313" deployment/\n'})}),"\n",(0,o.jsxs)(t.p,{children:["The Contrast Coordinator will rotate the mesh ca certificate on the manifest update. Workload certificates issued\nafter the manifest are thus issued by another certificate authority and services receiving the new CA certificate chain\nwon't trust parts of the deployment that got their certificate issued before the update. This way, Contrast ensures\nthat parts of the deployment that received a security update won't be infected by parts of the deployment at an older\npatch level that may have been compromised. The ",(0,o.jsx)(t.code,{children:"mesh-ca.pem"})," is updated with the new CA certificate chain."]}),"\n",(0,o.jsx)(t.h3,{id:"rolling-out-the-update",children:"Rolling out the update"}),"\n",(0,o.jsx)(t.p,{children:"The Coordinator has the new manifest set, but the different containers of the app are still\nusing the older certificate authority. The Contrast Initializer terminates after the initial attestation\nflow and won't pull new certificates on manifest updates."}),"\n",(0,o.jsx)(t.p,{children:"To roll out the update, use:"}),"\n",(0,o.jsx)(t.pre,{children:(0,o.jsx)(t.code,{className:"language-sh",children:"kubectl rollout restart deployment/emoji\nkubectl rollout restart deployment/vote-bot\nkubectl rollout restart deployment/voting\nkubectl rollout restart deployment/web\n"})}),"\n",(0,o.jsx)(t.p,{children:"After the update has been rolled out, connecting to the frontend using curl will successfully validate\nthe service certificate and return the HTML document of the voting site:"}),"\n",(0,o.jsx)(t.pre,{children:(0,o.jsx)(t.code,{className:"language-sh",children:'curl --cacert ./mesh-ca.pem "https://${frontendIP}:443"\n'})})]})}function h(e={}){const{wrapper:t}={...(0,i.R)(),...e.components};return t?(0,o.jsx)(t,{...e,children:(0,o.jsx)(c,{...e})}):c(e)}},2866:(e,t,n)=>{n.d(t,{A:()=>o});const o=n.p+"assets/images/emoijvoto-3fb48da575d6d4adf76cc90caa35d762.png"},8453:(e,t,n)=>{n.d(t,{R:()=>a,x:()=>r});var o=n(6540);const i={},s=o.createContext(i);function a(e){const t=o.useContext(s);return o.useMemo((function(){return"function"==typeof e?e(t):{...t,...e}}),[t,e])}function r(e){let t;return t=e.disableParentContext?"function"==typeof e.components?e.components(i):e.components||i:a(e.components),o.createElement(s.Provider,{value:t},e.children)}}}]);