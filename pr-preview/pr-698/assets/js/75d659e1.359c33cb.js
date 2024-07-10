"use strict";(self.webpackChunkcontrast_docs=self.webpackChunkcontrast_docs||[]).push([[8597],{1809:(e,n,t)=>{t.r(n),t.d(n,{assets:()=>c,contentTitle:()=>o,default:()=>h,frontMatter:()=>i,metadata:()=>a,toc:()=>l});var r=t(4848),s=t(8453);const i={},o="Workload deployment",a={id:"deployment",title:"Workload deployment",description:"The following instructions will guide you through the process of making an existing Kubernetes deployment",source:"@site/versioned_docs/version-0.6/deployment.md",sourceDirName:".",slug:"/deployment",permalink:"/contrast/pr-preview/pr-698/0.6/deployment",draft:!1,unlisted:!1,editUrl:"https://github.com/edgelesssys/contrast/edit/main/docs/versioned_docs/version-0.6/deployment.md",tags:[],version:"0.6",frontMatter:{},sidebar:"docs",previous:{title:"Confidential emoji voting",permalink:"/contrast/pr-preview/pr-698/0.6/examples/emojivoto"},next:{title:"Components",permalink:"/contrast/pr-preview/pr-698/0.6/components/"}},c={},l=[{value:"Deploy the Contrast runtime",id:"deploy-the-contrast-runtime",level:2},{value:"Deploy the Contrast Coordinator",id:"deploy-the-contrast-coordinator",level:2},{value:"Prepare your Kubernetes resources",id:"prepare-your-kubernetes-resources",level:2},{value:"RuntimeClass and Initializer",id:"runtimeclass-and-initializer",level:3},{value:"Handling TLS",id:"handling-tls",level:3},{value:"Generate policy annotations and manifest",id:"generate-policy-annotations-and-manifest",level:2},{value:"Apply the resources",id:"apply-the-resources",level:2},{value:"Connect to the Contrast Coordinator",id:"connect-to-the-contrast-coordinator",level:2},{value:"Set the manifest",id:"set-the-manifest",level:2},{value:"Verify the Coordinator",id:"verify-the-coordinator",level:2},{value:"Communicate with workloads",id:"communicate-with-workloads",level:2}];function d(e){const n={a:"a",admonition:"admonition",code:"code",h1:"h1",h2:"h2",h3:"h3",li:"li",p:"p",pre:"pre",ul:"ul",...(0,s.R)(),...e.components},{TabItem:t,Tabs:i}=n;return t||p("TabItem",!0),i||p("Tabs",!0),(0,r.jsxs)(r.Fragment,{children:[(0,r.jsx)(n.h1,{id:"workload-deployment",children:"Workload deployment"}),"\n",(0,r.jsx)(n.p,{children:"The following instructions will guide you through the process of making an existing Kubernetes deployment\nconfidential and deploying it together with Contrast."}),"\n",(0,r.jsxs)(n.p,{children:["A running CoCo-enabled cluster is required for these steps, see the ",(0,r.jsx)(n.a,{href:"/contrast/pr-preview/pr-698/0.6/getting-started/cluster-setup",children:"setup guide"})," on how to set it up."]}),"\n",(0,r.jsx)(n.h2,{id:"deploy-the-contrast-runtime",children:"Deploy the Contrast runtime"}),"\n",(0,r.jsxs)(n.p,{children:["Contrast depends on a ",(0,r.jsxs)(n.a,{href:"/contrast/pr-preview/pr-698/0.6/components/runtime",children:["custom Kubernetes ",(0,r.jsx)(n.code,{children:"RuntimeClass"})," (",(0,r.jsx)(n.code,{children:"contrast-cc"}),")"]}),",\nwhich needs to be installed in the cluster prior to the Coordinator or any confidential workloads.\nThis consists of a ",(0,r.jsx)(n.code,{children:"RuntimeClass"})," resource and a ",(0,r.jsx)(n.code,{children:"DaemonSet"})," that performs installation on worker nodes.\nThis step is only required once for each version of the runtime.\nIt can be shared between Contrast deployments."]}),"\n",(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-sh",children:"kubectl apply -f https://github.com/edgelesssys/contrast/releases/latest/download/runtime.yml\n"})}),"\n",(0,r.jsx)(n.h2,{id:"deploy-the-contrast-coordinator",children:"Deploy the Contrast Coordinator"}),"\n",(0,r.jsx)(n.p,{children:"Install the latest Contrast Coordinator release, comprising a single replica deployment and a\nLoadBalancer service, into your cluster."}),"\n",(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-sh",children:"kubectl apply -f https://github.com/edgelesssys/contrast/releases/latest/download/coordinator.yml\n"})}),"\n",(0,r.jsx)(n.h2,{id:"prepare-your-kubernetes-resources",children:"Prepare your Kubernetes resources"}),"\n",(0,r.jsx)(n.p,{children:"Your Kubernetes resources need some modifications to run as Confidential Containers.\nThis section guides you through the process and outlines the necessary changes."}),"\n",(0,r.jsx)(n.h3,{id:"runtimeclass-and-initializer",children:"RuntimeClass and Initializer"}),"\n",(0,r.jsx)(n.p,{children:"Contrast will add annotations to your Kubernetes YAML files. If you want to keep the original files\nunchanged, you can copy the files into a separate local directory.\nYou can also generate files from a Helm chart or from a Kustomization."}),"\n",(0,r.jsxs)(i,{groupId:"yaml-source",children:[(0,r.jsx)(t,{value:"kustomize",label:"kustomize",children:(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-sh",children:"mkdir resources\nkustomize build $MY_RESOURCE_DIR > resources/all.yml\n"})})}),(0,r.jsx)(t,{value:"helm",label:"helm",children:(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-sh",children:"mkdir resources\nhelm template $RELEASE_NAME $CHART_NAME > resources/all.yml\n"})})}),(0,r.jsx)(t,{value:"copy",label:"copy",children:(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-sh",children:"cp -R $MY_RESOURCE_DIR resources/\n"})})})]}),"\n",(0,r.jsxs)(n.p,{children:["To specify that a workload (pod, deployment, etc.) should be deployed as confidential containers,\nadd ",(0,r.jsx)(n.code,{children:"runtimeClassName: contrast-cc"})," to the pod spec (pod definition or template).\nThis is a placeholder name that will be replaced by a versioned ",(0,r.jsx)(n.code,{children:"runtimeClassName"})," when generating policies.\nIn addition, add the Contrast Initializer as ",(0,r.jsx)(n.code,{children:"initContainers"})," to these workloads and configure the\nworkload to use the certificates written to a ",(0,r.jsx)(n.code,{children:"volumeMount"})," named ",(0,r.jsx)(n.code,{children:"tls-certs"}),"."]}),"\n",(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-yaml",children:'spec: # v1.PodSpec\n  runtimeClassName: contrast-cc\n  initContainers:\n  - name: initializer\n    image: "ghcr.io/edgelesssys/contrast/initializer:latest"\n    env:\n    - name: COORDINATOR_HOST\n      value: coordinator\n    volumeMounts:\n    - name: tls-certs\n      mountPath: /tls-config\n  volumes:\n  - name: tls-certs\n    emptyDir: {}\n'})}),"\n",(0,r.jsx)(n.h3,{id:"handling-tls",children:"Handling TLS"}),"\n",(0,r.jsxs)(n.p,{children:["The initializer populates the shared volume with X.509 certificates for your workload.\nThese certificates are used by the ",(0,r.jsx)(n.a,{href:"/contrast/pr-preview/pr-698/0.6/components/service-mesh",children:"Contrast Service Mesh"}),", but can also be used by your application directly.\nThe following tab group explains the setup for both scenarios."]}),"\n",(0,r.jsxs)(i,{groupId:"tls",children:[(0,r.jsxs)(t,{value:"mesh",label:"Drop-in service mesh",children:[(0,r.jsx)(n.p,{children:"Contrast can be configured to handle TLS in a sidecar container.\nThis is useful for workloads that are hard to configure with custom certificates, like Java applications."}),(0,r.jsx)(n.p,{children:"Configuration of the sidecar depends heavily on the application.\nThe following example is for an application with these properties:"}),(0,r.jsxs)(n.ul,{children:["\n",(0,r.jsx)(n.li,{children:"The app has a main application at TCP port 8001, which should be TLS-wrapped and doesn't require client authentication."}),"\n",(0,r.jsx)(n.li,{children:"The app has a metrics endpoint at TCP port 8080, which should be accessible in plain text."}),"\n",(0,r.jsx)(n.li,{children:"All other endpoints require client authentication."}),"\n",(0,r.jsxs)(n.li,{children:["The app connects to a Kubernetes service ",(0,r.jsx)(n.code,{children:"backend.default:4001"}),", which requires client authentication."]}),"\n"]}),(0,r.jsx)(n.p,{children:"Add the following sidecar definition to your workload:"}),(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-yaml",children:'spec: # v1.PodSpec\n  initContainers:\n  - name: tls-sidecar\n    image: "ghcr.io/edgelesssys/contrast/service-mesh-proxy:latest"\n    restartPolicy: Always\n    env:\n    - name: EDG_INGRESS_PROXY_CONFIG\n      value: "main#8001#false##metrics#8080#true"\n    - name: EDG_EGRESS_PROXY_CONFIG\n      value: "backend#127.0.0.2:4001#backend.default:4001"\n    volumeMounts:\n    - name: tls-certs\n      mountPath: /tls-config\n'})}),(0,r.jsxs)(n.p,{children:["The only change required to the app itself is to let it connect to ",(0,r.jsx)(n.code,{children:"127.0.0.2:4001"})," to reach the backend service.\nYou can find more detailed documentation in the ",(0,r.jsx)(n.a,{href:"/contrast/pr-preview/pr-698/0.6/components/service-mesh",children:"Service Mesh chapter"}),"."]})]}),(0,r.jsxs)(t,{value:"go",label:"Go integration",children:[(0,r.jsxs)(n.p,{children:["The mesh certificate contained in ",(0,r.jsx)(n.code,{children:"certChain.pem"})," authenticates this workload, while the mesh CA certificate ",(0,r.jsx)(n.code,{children:"mesh-ca.pem"})," authenticates its peers.\nYour app should turn on client authentication to ensure peers are running as confidential containers, too.\nSee the ",(0,r.jsx)(n.a,{href:"/contrast/pr-preview/pr-698/0.6/architecture/certificates",children:"Certificate Authority"})," section for detailed information about these certificates."]}),(0,r.jsx)(n.p,{children:"The following example shows how to configure a Golang app, with error handling omitted for clarity."}),(0,r.jsxs)(i,{groupId:"golang-tls-setup",children:[(0,r.jsx)(t,{value:"client",label:"Client",children:(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-go",children:'caCerts := x509.NewCertPool()\ncaCert, _ := os.ReadFile("/tls-config/mesh-ca.pem")\ncaCerts.AppendCertsFromPEM(caCert)\ncert, _ := tls.LoadX509KeyPair("/tls-config/certChain.pem", "/tls-config/key.pem")\ncfg := &tls.Config{\n  Certificates: []tls.Certificate{cert},\n  RootCAs: caCerts,\n}\n'})})}),(0,r.jsx)(t,{value:"server",label:"Server",children:(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-go",children:'caCerts := x509.NewCertPool()\ncaCert, _ := os.ReadFile("/tls-config/mesh-ca.pem")\ncaCerts.AppendCertsFromPEM(caCert)\ncert, _ := tls.LoadX509KeyPair("/tls-config/certChain.pem", "/tls-config/key.pem")\ncfg := &tls.Config{\n  Certificates: []tls.Certificate{cert},\n  ClientAuth: tls.RequireAndVerifyClientCert,\n  ClientCAs: caCerts,\n}\n'})})})]})]})]}),"\n",(0,r.jsx)(n.h2,{id:"generate-policy-annotations-and-manifest",children:"Generate policy annotations and manifest"}),"\n",(0,r.jsxs)(n.p,{children:["Run the ",(0,r.jsx)(n.code,{children:"generate"})," command to generate the execution policies and add them as annotations to your\ndeployment files. A ",(0,r.jsx)(n.code,{children:"manifest.json"})," with the reference values of your deployment will be created."]}),"\n",(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-sh",children:"contrast generate resources/\n"})}),"\n",(0,r.jsx)(n.admonition,{type:"warning",children:(0,r.jsxs)(n.p,{children:["Please be aware that runtime policies currently have some blind spots. For example, they can't guarantee the starting order of containers. See the ",(0,r.jsx)(n.a,{href:"/contrast/pr-preview/pr-698/0.6/known-limitations#runtime-policies",children:"current limitations"})," for more details."]})}),"\n",(0,r.jsx)(n.h2,{id:"apply-the-resources",children:"Apply the resources"}),"\n",(0,r.jsx)(n.p,{children:"Apply the resources to the cluster. Your workloads will block in the initialization phase until a\nmanifest is set at the Coordinator."}),"\n",(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-sh",children:"kubectl apply -f resources/\n"})}),"\n",(0,r.jsx)(n.h2,{id:"connect-to-the-contrast-coordinator",children:"Connect to the Contrast Coordinator"}),"\n",(0,r.jsx)(n.p,{children:"For the next steps, we will need to connect to the Coordinator. The released Coordinator resource\nincludes a LoadBalancer definition we can use."}),"\n",(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-sh",children:"coordinator=$(kubectl get svc coordinator -o=jsonpath='{.status.loadBalancer.ingress[0].ip}')\n"})}),"\n",(0,r.jsxs)(n.admonition,{title:"Port-forwarding of Confidential Containers",type:"info",children:[(0,r.jsxs)(n.p,{children:[(0,r.jsx)(n.code,{children:"kubectl port-forward"})," uses a Container Runtime Interface (CRI) method that isn't supported by the Kata shim.\nIf you can't use a public load balancer, you can deploy a ",(0,r.jsx)(n.a,{href:"https://github.com/edgelesssys/contrast/blob/ddc371b/deployments/emojivoto/portforwarder.yml",children:"port-forwarder"}),".\nThe port-forwarder relays traffic from a CoCo pod and can be accessed via ",(0,r.jsx)(n.code,{children:"kubectl port-forward"}),"."]}),(0,r.jsxs)(n.p,{children:["Upstream tracking issue: ",(0,r.jsx)(n.a,{href:"https://github.com/kata-containers/kata-containers/issues/1693",children:"https://github.com/kata-containers/kata-containers/issues/1693"}),"."]})]}),"\n",(0,r.jsx)(n.h2,{id:"set-the-manifest",children:"Set the manifest"}),"\n",(0,r.jsx)(n.p,{children:"Attest the Coordinator and set the manifest:"}),"\n",(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-sh",children:'contrast set -c "${coordinator}:1313" resources/\n'})}),"\n",(0,r.jsx)(n.p,{children:"After this step, the Coordinator will start issuing TLS certs to the workloads. The init container\nwill fetch a certificate for the workload and the workload is started."}),"\n",(0,r.jsx)(n.h2,{id:"verify-the-coordinator",children:"Verify the Coordinator"}),"\n",(0,r.jsxs)(n.p,{children:["An end user (data owner) can verify the Contrast deployment using the ",(0,r.jsx)(n.code,{children:"verify"})," command."]}),"\n",(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-sh",children:'contrast verify -c "${coordinator}:1313"\n'})}),"\n",(0,r.jsxs)(n.p,{children:["The CLI will attest the Coordinator using embedded reference values. The CLI will write the service mesh\nroot certificate and the history of manifests into the ",(0,r.jsx)(n.code,{children:"verify/"})," directory. In addition, the policies referenced\nin the manifest are also written to the directory."]}),"\n",(0,r.jsx)(n.h2,{id:"communicate-with-workloads",children:"Communicate with workloads"}),"\n",(0,r.jsxs)(n.p,{children:["You can securely connect to the workloads using the Coordinator's ",(0,r.jsx)(n.code,{children:"mesh-ca.pem"})," as a trusted CA certificate.\nFirst, expose the service on a public IP address via a LoadBalancer service:"]}),"\n",(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-sh",children:"kubectl patch svc ${MY_SERVICE} -p '{\"spec\": {\"type\": \"LoadBalancer\"}}'\nkubectl wait --timeout=30s --for=jsonpath='{.status.loadBalancer.ingress}' service/${MY_SERVICE}\nlbip=$(kubectl get svc ${MY_SERVICE} -o=jsonpath='{.status.loadBalancer.ingress[0].ip}')\necho $lbip\n"})}),"\n",(0,r.jsxs)(n.admonition,{title:"Subject alternative names and LoadBalancer IP",type:"info",children:[(0,r.jsx)(n.p,{children:"By default, mesh certificates are issued with a wildcard DNS entry. The web frontend is accessed\nvia load balancer IP in this demo. Tools like curl check the certificate for IP entries in the SAN field.\nValidation fails since the certificate contains no IP entries as a subject alternative name (SAN).\nFor example, a connection attempt using the curl and the mesh CA certificate with throw the following error:"}),(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-sh",children:"$ curl --cacert ./verify/mesh-ca.pem \"https://${frontendIP}:443\"\ncurl: (60) SSL: no alternative certificate subject name matches target host name '203.0.113.34'\n"})})]}),"\n",(0,r.jsxs)(n.p,{children:["Using ",(0,r.jsx)(n.code,{children:"openssl"}),", the certificate of the service can be validated with the ",(0,r.jsx)(n.code,{children:"mesh-ca.pem"}),":"]}),"\n",(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-sh",children:"openssl s_client -CAfile verify/mesh-ca.pem -verify_return_error -connect ${frontendIP}:443 < /dev/null\n"})})]})}function h(e={}){const{wrapper:n}={...(0,s.R)(),...e.components};return n?(0,r.jsx)(n,{...e,children:(0,r.jsx)(d,{...e})}):d(e)}function p(e,n){throw new Error("Expected "+(n?"component":"object")+" `"+e+"` to be defined: you likely forgot to import, pass, or provide it.")}},8453:(e,n,t)=>{t.d(n,{R:()=>o,x:()=>a});var r=t(6540);const s={},i=r.createContext(s);function o(e){const n=r.useContext(i);return r.useMemo((function(){return"function"==typeof e?e(n):{...n,...e}}),[n,e])}function a(e){let n;return n=e.disableParentContext?"function"==typeof e.components?e.components(s):e.components||s:o(e.components),r.createElement(i.Provider,{value:n},e.children)}}}]);