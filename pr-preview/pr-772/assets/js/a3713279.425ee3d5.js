"use strict";(self.webpackChunkcontrast_docs=self.webpackChunkcontrast_docs||[]).push([[9588],{4854:(e,t,n)=>{n.r(t),n.d(t,{assets:()=>c,contentTitle:()=>i,default:()=>h,frontMatter:()=>s,metadata:()=>a,toc:()=>l});var r=n(4848),o=n(8453);const s={},i="Workload deployment",a={id:"deployment",title:"Workload deployment",description:"The following instructions will guide you through the process of making an existing Kubernetes deployment",source:"@site/docs/deployment.md",sourceDirName:".",slug:"/deployment",permalink:"/contrast/pr-preview/pr-772/next/deployment",draft:!1,unlisted:!1,editUrl:"https://github.com/edgelesssys/contrast/edit/main/docs/docs/deployment.md",tags:[],version:"current",frontMatter:{},sidebar:"docs",previous:{title:"Confidential emoji voting",permalink:"/contrast/pr-preview/pr-772/next/examples/emojivoto"},next:{title:"Troubleshooting",permalink:"/contrast/pr-preview/pr-772/next/troubleshooting"}},c={},l=[{value:"Deploy the Contrast runtime",id:"deploy-the-contrast-runtime",level:2},{value:"Deploy the Contrast Coordinator",id:"deploy-the-contrast-coordinator",level:2},{value:"Prepare your Kubernetes resources",id:"prepare-your-kubernetes-resources",level:2},{value:"RuntimeClass",id:"runtimeclass",level:3},{value:"Handling TLS",id:"handling-tls",level:3},{value:"Generate policy annotations and manifest",id:"generate-policy-annotations-and-manifest",level:2},{value:"Apply the resources",id:"apply-the-resources",level:2},{value:"Connect to the Contrast Coordinator",id:"connect-to-the-contrast-coordinator",level:2},{value:"Set the manifest",id:"set-the-manifest",level:2},{value:"Verify the Coordinator",id:"verify-the-coordinator",level:2},{value:"Communicate with workloads",id:"communicate-with-workloads",level:2},{value:"Recover the Coordinator",id:"recover-the-coordinator",level:2}];function d(e){const t={a:"a",admonition:"admonition",code:"code",h1:"h1",h2:"h2",h3:"h3",li:"li",p:"p",pre:"pre",ul:"ul",...(0,o.R)(),...e.components},{TabItem:n,Tabs:s}=t;return n||p("TabItem",!0),s||p("Tabs",!0),(0,r.jsxs)(r.Fragment,{children:[(0,r.jsx)(t.h1,{id:"workload-deployment",children:"Workload deployment"}),"\n",(0,r.jsx)(t.p,{children:"The following instructions will guide you through the process of making an existing Kubernetes deployment\nconfidential and deploying it together with Contrast."}),"\n",(0,r.jsxs)(t.p,{children:["A running CoCo-enabled cluster is required for these steps, see the ",(0,r.jsx)(t.a,{href:"/contrast/pr-preview/pr-772/next/getting-started/cluster-setup",children:"setup guide"})," on how to set it up."]}),"\n",(0,r.jsx)(t.h2,{id:"deploy-the-contrast-runtime",children:"Deploy the Contrast runtime"}),"\n",(0,r.jsxs)(t.p,{children:["Contrast depends on a ",(0,r.jsxs)(t.a,{href:"/contrast/pr-preview/pr-772/next/components/runtime",children:["custom Kubernetes ",(0,r.jsx)(t.code,{children:"RuntimeClass"})," (",(0,r.jsx)(t.code,{children:"contrast-cc"}),")"]}),",\nwhich needs to be installed in the cluster prior to the Coordinator or any confidential workloads.\nThis consists of a ",(0,r.jsx)(t.code,{children:"RuntimeClass"})," resource and a ",(0,r.jsx)(t.code,{children:"DaemonSet"})," that performs installation on worker nodes.\nThis step is only required once for each version of the runtime.\nIt can be shared between Contrast deployments."]}),"\n",(0,r.jsx)(t.pre,{children:(0,r.jsx)(t.code,{className:"language-sh",children:"kubectl apply -f https://github.com/edgelesssys/contrast/releases/latest/download/runtime.yml\n"})}),"\n",(0,r.jsx)(t.h2,{id:"deploy-the-contrast-coordinator",children:"Deploy the Contrast Coordinator"}),"\n",(0,r.jsx)(t.p,{children:"Install the latest Contrast Coordinator release, comprising a single replica deployment and a\nLoadBalancer service, into your cluster."}),"\n",(0,r.jsx)(t.pre,{children:(0,r.jsx)(t.code,{className:"language-sh",children:"kubectl apply -f https://github.com/edgelesssys/contrast/releases/latest/download/coordinator.yml\n"})}),"\n",(0,r.jsx)(t.h2,{id:"prepare-your-kubernetes-resources",children:"Prepare your Kubernetes resources"}),"\n",(0,r.jsx)(t.p,{children:"Your Kubernetes resources need some modifications to run as Confidential Containers.\nThis section guides you through the process and outlines the necessary changes."}),"\n",(0,r.jsx)(t.h3,{id:"runtimeclass",children:"RuntimeClass"}),"\n",(0,r.jsx)(t.p,{children:"Contrast will add annotations to your Kubernetes YAML files. If you want to keep the original files\nunchanged, you can copy the files into a separate local directory.\nYou can also generate files from a Helm chart or from a Kustomization."}),"\n",(0,r.jsxs)(s,{groupId:"yaml-source",children:[(0,r.jsx)(n,{value:"kustomize",label:"kustomize",children:(0,r.jsx)(t.pre,{children:(0,r.jsx)(t.code,{className:"language-sh",children:"mkdir resources\nkustomize build $MY_RESOURCE_DIR > resources/all.yml\n"})})}),(0,r.jsx)(n,{value:"helm",label:"helm",children:(0,r.jsx)(t.pre,{children:(0,r.jsx)(t.code,{className:"language-sh",children:"mkdir resources\nhelm template $RELEASE_NAME $CHART_NAME > resources/all.yml\n"})})}),(0,r.jsx)(n,{value:"copy",label:"copy",children:(0,r.jsx)(t.pre,{children:(0,r.jsx)(t.code,{className:"language-sh",children:"cp -R $MY_RESOURCE_DIR resources/\n"})})})]}),"\n",(0,r.jsxs)(t.p,{children:["To specify that a workload (pod, deployment, etc.) should be deployed as confidential containers,\nadd ",(0,r.jsx)(t.code,{children:"runtimeClassName: contrast-cc"})," to the pod spec (pod definition or template).\nThis is a placeholder name that will be replaced by a versioned ",(0,r.jsx)(t.code,{children:"runtimeClassName"})," when generating policies."]}),"\n",(0,r.jsx)(t.pre,{children:(0,r.jsx)(t.code,{className:"language-yaml",children:"spec: # v1.PodSpec\n  runtimeClassName: contrast-cc\n"})}),"\n",(0,r.jsx)(t.h3,{id:"handling-tls",children:"Handling TLS"}),"\n",(0,r.jsxs)(t.p,{children:["In the initialization process, the ",(0,r.jsx)(t.code,{children:"contrast-tls-certs"})," shared volume is populated with X.509 certificates for your workload.\nThese certificates are used by the ",(0,r.jsx)(t.a,{href:"/contrast/pr-preview/pr-772/next/components/service-mesh",children:"Contrast Service Mesh"}),", but can also be used by your application directly.\nThe following tab group explains the setup for both scenarios."]}),"\n",(0,r.jsxs)(s,{groupId:"tls",children:[(0,r.jsxs)(n,{value:"mesh",label:"Drop-in service mesh",children:[(0,r.jsx)(t.p,{children:"Contrast can be configured to handle TLS in a sidecar container.\nThis is useful for workloads that are hard to configure with custom certificates, like Java applications."}),(0,r.jsx)(t.p,{children:"Configuration of the sidecar depends heavily on the application.\nThe following example is for an application with these properties:"}),(0,r.jsxs)(t.ul,{children:["\n",(0,r.jsx)(t.li,{children:"The container has a main application at TCP port 8001, which should be TLS-wrapped and doesn't require client authentication."}),"\n",(0,r.jsx)(t.li,{children:"The container has a metrics endpoint at TCP port 8080, which should be accessible in plain text."}),"\n",(0,r.jsx)(t.li,{children:"All other endpoints require client authentication."}),"\n",(0,r.jsxs)(t.li,{children:["The app connects to a Kubernetes service ",(0,r.jsx)(t.code,{children:"backend.default:4001"}),", which requires client authentication."]}),"\n"]}),(0,r.jsx)(t.p,{children:"Add the following annotations to your workload:"}),(0,r.jsx)(t.pre,{children:(0,r.jsx)(t.code,{className:"language-yaml",children:'metadata: # apps/v1.Deployment, apps/v1.DaemonSet, ...\n  annotations:\n    contrast.edgeless.systems/servicemesh-ingress: "main#8001#false##metrics#8080#true"\n    contrast.edgeless.systems/servicemesh-egress: "backend#127.0.0.2:4001#backend.default:4001"\n'})}),(0,r.jsxs)(t.p,{children:["During the ",(0,r.jsx)(t.code,{children:"generate"})," step, this configuration will be translated into a Service Mesh sidecar container which handles TLS connections automatically.\nThe only change required to the app itself is to let it connect to ",(0,r.jsx)(t.code,{children:"127.0.0.2:4001"})," to reach the backend service.\nYou can find more detailed documentation in the ",(0,r.jsx)(t.a,{href:"/contrast/pr-preview/pr-772/next/components/service-mesh",children:"Service Mesh chapter"}),"."]})]}),(0,r.jsxs)(n,{value:"go",label:"Go integration",children:[(0,r.jsxs)(t.p,{children:["The mesh certificate contained in ",(0,r.jsx)(t.code,{children:"certChain.pem"})," authenticates this workload, while the mesh CA certificate ",(0,r.jsx)(t.code,{children:"mesh-ca.pem"})," authenticates its peers.\nYour app should turn on client authentication to ensure peers are running as confidential containers, too.\nSee the ",(0,r.jsx)(t.a,{href:"/contrast/pr-preview/pr-772/next/architecture/certificates",children:"Certificate Authority"})," section for detailed information about these certificates."]}),(0,r.jsx)(t.p,{children:"The following example shows how to configure a Golang app, with error handling omitted for clarity."}),(0,r.jsxs)(s,{groupId:"golang-tls-setup",children:[(0,r.jsx)(n,{value:"client",label:"Client",children:(0,r.jsx)(t.pre,{children:(0,r.jsx)(t.code,{className:"language-go",children:'caCerts := x509.NewCertPool()\ncaCert, _ := os.ReadFile("/tls-config/mesh-ca.pem")\ncaCerts.AppendCertsFromPEM(caCert)\ncert, _ := tls.LoadX509KeyPair("/tls-config/certChain.pem", "/tls-config/key.pem")\ncfg := &tls.Config{\n  Certificates: []tls.Certificate{cert},\n  RootCAs: caCerts,\n}\n'})})}),(0,r.jsx)(n,{value:"server",label:"Server",children:(0,r.jsx)(t.pre,{children:(0,r.jsx)(t.code,{className:"language-go",children:'caCerts := x509.NewCertPool()\ncaCert, _ := os.ReadFile("/tls-config/mesh-ca.pem")\ncaCerts.AppendCertsFromPEM(caCert)\ncert, _ := tls.LoadX509KeyPair("/tls-config/certChain.pem", "/tls-config/key.pem")\ncfg := &tls.Config{\n  Certificates: []tls.Certificate{cert},\n  ClientAuth: tls.RequireAndVerifyClientCert,\n  ClientCAs: caCerts,\n}\n'})})})]})]})]}),"\n",(0,r.jsx)(t.h2,{id:"generate-policy-annotations-and-manifest",children:"Generate policy annotations and manifest"}),"\n",(0,r.jsxs)(t.p,{children:["Run the ",(0,r.jsx)(t.code,{children:"generate"})," command to add the necessary components to your deployment files.\nThis will add the Contrast Initializer to every workload with the specified ",(0,r.jsx)(t.code,{children:"contrast-cc"})," runtime class\nand the Contrast Service Mesh to all workloads that have a specified configuration.\nAfter that, it will generate the execution policies and add them as annotations to your deployment files.\nA ",(0,r.jsx)(t.code,{children:"manifest.json"})," with the reference values of your deployment will be created."]}),"\n",(0,r.jsx)(t.pre,{children:(0,r.jsx)(t.code,{className:"language-sh",children:"contrast generate --reference-values aks-clh-snp resources/\n"})}),"\n",(0,r.jsx)(t.admonition,{type:"warning",children:(0,r.jsxs)(t.p,{children:["Please be aware that runtime policies currently have some blind spots. For example, they can't guarantee the starting order of containers. See the ",(0,r.jsx)(t.a,{href:"/contrast/pr-preview/pr-772/next/features-limitations#runtime-policies",children:"current limitations"})," for more details."]})}),"\n",(0,r.jsx)(t.p,{children:"If you don't want the Contrast Initializer to automatically be added to your\nworkloads, there are two ways you can skip the Initializer injection step,\ndepending on how you want to customize your deployment."}),"\n",(0,r.jsxs)(s,{groupId:"injection",children:[(0,r.jsxs)(n,{value:"flag",label:"Command-line flag",children:[(0,r.jsxs)(t.p,{children:["You can disable the Initializer injection completely by specifying the\n",(0,r.jsx)(t.code,{children:"--skip-initializer"})," flag in the ",(0,r.jsx)(t.code,{children:"generate"})," command."]}),(0,r.jsx)(t.pre,{children:(0,r.jsx)(t.code,{className:"language-sh",children:"contrast generate --reference-values aks-clh-snp --skip-initializer resources/\n"})})]}),(0,r.jsxs)(n,{value:"annotation",label:"Per-workload annotation",children:[(0,r.jsxs)(t.p,{children:["If you want to disable the Initializer injection for a specific workload with\nthe ",(0,r.jsx)(t.code,{children:"contrast-cc"})," runtime class, you can do so by adding an annotation to the workload."]}),(0,r.jsx)(t.pre,{children:(0,r.jsx)(t.code,{className:"language-yaml",children:'metadata: # apps/v1.Deployment, apps/v1.DaemonSet, ...\n  annotations:\n    contrast.edgeless.systems/skip-initializer: "true"\n'})})]})]}),"\n",(0,r.jsxs)(t.p,{children:["When disabling the automatic Initializer injection, you can manually add the\nInitializer as a sidecar container to your workload before generating the\npolicies. Configure the workload to use the certificates written to the\n",(0,r.jsx)(t.code,{children:"contrast-tls-certs"})," ",(0,r.jsx)(t.code,{children:"volumeMount"}),"."]}),"\n",(0,r.jsx)(t.pre,{children:(0,r.jsx)(t.code,{className:"language-yaml",children:'# v1.PodSpec\nspec:\n  initContainers:\n    - env:\n        - name: COORDINATOR_HOST\n          value: coordinator\n      image: "ghcr.io/edgelesssys/contrast/initializer:latest"\n      name: contrast-initializer\n      volumeMounts:\n        - mountPath: /tls-config\n          name: contrast-tls-certs\n  volumes:\n    - emptyDir: {}\n      name: contrast-tls-certs\n'})}),"\n",(0,r.jsx)(t.h2,{id:"apply-the-resources",children:"Apply the resources"}),"\n",(0,r.jsx)(t.p,{children:"Apply the resources to the cluster. Your workloads will block in the initialization phase until a\nmanifest is set at the Coordinator."}),"\n",(0,r.jsx)(t.pre,{children:(0,r.jsx)(t.code,{className:"language-sh",children:"kubectl apply -f resources/\n"})}),"\n",(0,r.jsx)(t.h2,{id:"connect-to-the-contrast-coordinator",children:"Connect to the Contrast Coordinator"}),"\n",(0,r.jsx)(t.p,{children:"For the next steps, we will need to connect to the Coordinator. The released Coordinator resource\nincludes a LoadBalancer definition we can use."}),"\n",(0,r.jsx)(t.pre,{children:(0,r.jsx)(t.code,{className:"language-sh",children:"coordinator=$(kubectl get svc coordinator -o=jsonpath='{.status.loadBalancer.ingress[0].ip}')\n"})}),"\n",(0,r.jsxs)(t.admonition,{title:"Port-forwarding of Confidential Containers",type:"info",children:[(0,r.jsxs)(t.p,{children:[(0,r.jsx)(t.code,{children:"kubectl port-forward"})," uses a Container Runtime Interface (CRI) method that isn't supported by the Kata shim.\nIf you can't use a public load balancer, you can deploy a ",(0,r.jsx)(t.a,{href:"https://github.com/edgelesssys/contrast/blob/ddc371b/deployments/emojivoto/portforwarder.yml",children:"port-forwarder"}),".\nThe port-forwarder relays traffic from a CoCo pod and can be accessed via ",(0,r.jsx)(t.code,{children:"kubectl port-forward"}),"."]}),(0,r.jsxs)(t.p,{children:["Upstream tracking issue: ",(0,r.jsx)(t.a,{href:"https://github.com/kata-containers/kata-containers/issues/1693",children:"https://github.com/kata-containers/kata-containers/issues/1693"}),"."]})]}),"\n",(0,r.jsx)(t.h2,{id:"set-the-manifest",children:"Set the manifest"}),"\n",(0,r.jsx)(t.p,{children:"Attest the Coordinator and set the manifest:"}),"\n",(0,r.jsx)(t.pre,{children:(0,r.jsx)(t.code,{className:"language-sh",children:'contrast set -c "${coordinator}:1313" resources/\n'})}),"\n",(0,r.jsx)(t.p,{children:"This will use the reference values from the manifest file to attest the Coordinator.\nAfter this step, the Coordinator will start issuing TLS certificates to the workloads. The init container\nwill fetch a certificate for the workload and the workload is started."}),"\n",(0,r.jsx)(t.h2,{id:"verify-the-coordinator",children:"Verify the Coordinator"}),"\n",(0,r.jsxs)(t.p,{children:["An end user (data owner) can verify the Contrast deployment using the ",(0,r.jsx)(t.code,{children:"verify"})," command."]}),"\n",(0,r.jsx)(t.pre,{children:(0,r.jsx)(t.code,{className:"language-sh",children:'contrast verify -c "${coordinator}:1313"\n'})}),"\n",(0,r.jsxs)(t.p,{children:["The CLI will attest the Coordinator using the reference values from the given manifest file. It will then write the\nservice mesh root certificate and the history of manifests into the ",(0,r.jsx)(t.code,{children:"verify/"})," directory. In addition, the policies\nreferenced in the active manifest are also written to the directory. The verification will fail if the active\nmanifest at the Coordinator doesn't match the manifest passed to the CLI."]}),"\n",(0,r.jsx)(t.h2,{id:"communicate-with-workloads",children:"Communicate with workloads"}),"\n",(0,r.jsxs)(t.p,{children:["You can securely connect to the workloads using the Coordinator's ",(0,r.jsx)(t.code,{children:"mesh-ca.pem"})," as a trusted CA certificate.\nFirst, expose the service on a public IP address via a LoadBalancer service:"]}),"\n",(0,r.jsx)(t.pre,{children:(0,r.jsx)(t.code,{className:"language-sh",children:"kubectl patch svc ${MY_SERVICE} -p '{\"spec\": {\"type\": \"LoadBalancer\"}}'\nkubectl wait --timeout=30s --for=jsonpath='{.status.loadBalancer.ingress}' service/${MY_SERVICE}\nlbip=$(kubectl get svc ${MY_SERVICE} -o=jsonpath='{.status.loadBalancer.ingress[0].ip}')\necho $lbip\n"})}),"\n",(0,r.jsxs)(t.admonition,{title:"Subject alternative names and LoadBalancer IP",type:"info",children:[(0,r.jsx)(t.p,{children:"By default, mesh certificates are issued with a wildcard DNS entry. The web frontend is accessed\nvia load balancer IP in this demo. Tools like curl check the certificate for IP entries in the SAN field.\nValidation fails since the certificate contains no IP entries as a subject alternative name (SAN).\nFor example, a connection attempt using the curl and the mesh CA certificate with throw the following error:"}),(0,r.jsx)(t.pre,{children:(0,r.jsx)(t.code,{className:"language-sh",children:"$ curl --cacert ./verify/mesh-ca.pem \"https://${frontendIP}:443\"\ncurl: (60) SSL: no alternative certificate subject name matches target host name '203.0.113.34'\n"})})]}),"\n",(0,r.jsxs)(t.p,{children:["Using ",(0,r.jsx)(t.code,{children:"openssl"}),", the certificate of the service can be validated with the ",(0,r.jsx)(t.code,{children:"mesh-ca.pem"}),":"]}),"\n",(0,r.jsx)(t.pre,{children:(0,r.jsx)(t.code,{className:"language-sh",children:"openssl s_client -CAfile verify/mesh-ca.pem -verify_return_error -connect ${frontendIP}:443 < /dev/null\n"})}),"\n",(0,r.jsx)(t.h2,{id:"recover-the-coordinator",children:"Recover the Coordinator"}),"\n",(0,r.jsx)(t.p,{children:"If the Contrast Coordinator restarts, it enters recovery mode and waits for an operator to provide key material.\nFor demonstration purposes, you can simulate this scenario by deleting the Coordinator pod."}),"\n",(0,r.jsx)(t.pre,{children:(0,r.jsx)(t.code,{className:"language-sh",children:"kubectl delete pod -l app.kubernetes.io/name=coordinator\n"})}),"\n",(0,r.jsxs)(t.p,{children:["Kubernetes schedules a new pod, but that pod doesn't have access to the key material the previous pod held in memory and can't issue certificates for workloads yet.\nYou can confirm this by running ",(0,r.jsx)(t.code,{children:"verify"})," again, or you can restart a workload pod, which should stay in the initialization phase.\nHowever, the secret seed in your working directory is sufficient to recover the coordinator."]}),"\n",(0,r.jsx)(t.pre,{children:(0,r.jsx)(t.code,{className:"language-sh",children:'contrast recover -c "${coordinator}:1313"\n'})}),"\n",(0,r.jsx)(t.p,{children:"Now that the Coordinator is recovered, all workloads should pass initialization and enter the running state.\nYou can now verify the Coordinator again, which should return the same manifest you set before."}),"\n",(0,r.jsx)(t.admonition,{type:"warning",children:(0,r.jsx)(t.p,{children:"The recovery process invalidates the mesh CA certificate:\nexisting workloads won't be able to communicate with workloads newly spawned.\nAll workloads should be restarted after the recovery succeeded."})})]})}function h(e={}){const{wrapper:t}={...(0,o.R)(),...e.components};return t?(0,r.jsx)(t,{...e,children:(0,r.jsx)(d,{...e})}):d(e)}function p(e,t){throw new Error("Expected "+(t?"component":"object")+" `"+e+"` to be defined: you likely forgot to import, pass, or provide it.")}},8453:(e,t,n)=>{n.d(t,{R:()=>i,x:()=>a});var r=n(6540);const o={},s=r.createContext(o);function i(e){const t=r.useContext(s);return r.useMemo((function(){return"function"==typeof e?e(t):{...t,...e}}),[t,e])}function a(e){let t;return t=e.disableParentContext?"function"==typeof e.components?e.components(o):e.components||o:i(e.components),r.createElement(s.Provider,{value:t},e.children)}}}]);