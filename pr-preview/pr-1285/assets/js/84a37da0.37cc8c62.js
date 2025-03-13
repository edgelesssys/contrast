"use strict";(self.webpackChunkcontrast_docs=self.webpackChunkcontrast_docs||[]).push([[4625],{48947:(e,n,t)=>{t.r(n),t.d(n,{assets:()=>l,contentTitle:()=>o,default:()=>h,frontMatter:()=>a,metadata:()=>s,toc:()=>c});const s=JSON.parse('{"id":"deployment","title":"Workload deployment","description":"The following instructions will guide you through the process of making an existing Kubernetes deployment","source":"@site/versioned_docs/version-1.5/deployment.md","sourceDirName":".","slug":"/deployment","permalink":"/contrast/pr-preview/pr-1285/1.5/deployment","draft":false,"unlisted":false,"editUrl":"https://github.com/edgelesssys/contrast/edit/main/docs/versioned_docs/version-1.5/deployment.md","tags":[],"version":"1.5","frontMatter":{},"sidebar":"docs","previous":{"title":"Encrypted volume mount","permalink":"/contrast/pr-preview/pr-1285/1.5/examples/mysql"},"next":{"title":"Troubleshooting","permalink":"/contrast/pr-preview/pr-1285/1.5/troubleshooting"}}');var r=t(74848),i=t(28453);const a={},o="Workload deployment",l={},c=[{value:"Deploy the Contrast runtime",id:"deploy-the-contrast-runtime",level:2},{value:"Deploy the Contrast Coordinator",id:"deploy-the-contrast-coordinator",level:2},{value:"Prepare your Kubernetes resources",id:"prepare-your-kubernetes-resources",level:2},{value:"Security review",id:"security-review",level:3},{value:"RuntimeClass",id:"runtimeclass",level:3},{value:"Pod resources",id:"pod-resources",level:3},{value:"Handling TLS",id:"handling-tls",level:3},{value:"Using GPUs",id:"using-gpus",level:3},{value:"Generate policy annotations and manifest",id:"generate-policy-annotations-and-manifest",level:2},{value:"Apply the resources",id:"apply-the-resources",level:2},{value:"Connect to the Contrast Coordinator",id:"connect-to-the-contrast-coordinator",level:2},{value:"Set the manifest",id:"set-the-manifest",level:2},{value:"Verify the Coordinator",id:"verify-the-coordinator",level:2},{value:"Communicate with workloads",id:"communicate-with-workloads",level:2},{value:"Recover the Coordinator",id:"recover-the-coordinator",level:2}];function d(e){const n={a:"a",admonition:"admonition",code:"code",em:"em",h1:"h1",h2:"h2",h3:"h3",header:"header",li:"li",p:"p",pre:"pre",ul:"ul",...(0,i.R)(),...e.components},{TabItem:t,Tabs:s}=n;return t||u("TabItem",!0),s||u("Tabs",!0),(0,r.jsxs)(r.Fragment,{children:[(0,r.jsx)(n.header,{children:(0,r.jsx)(n.h1,{id:"workload-deployment",children:"Workload deployment"})}),"\n",(0,r.jsx)(n.p,{children:"The following instructions will guide you through the process of making an existing Kubernetes deployment\nconfidential and deploying it together with Contrast."}),"\n",(0,r.jsxs)(s,{queryString:"platform",children:[(0,r.jsx)(t,{value:"aks-clh-snp",label:"AKS",default:!0,children:(0,r.jsxs)(n.p,{children:["A running CoCo-enabled cluster is required for these steps, see the ",(0,r.jsx)(n.a,{href:"/contrast/pr-preview/pr-1285/1.5/getting-started/cluster-setup",children:"setup guide"})," on how to set up a cluster on AKS."]})}),(0,r.jsx)(t,{value:"k3s-qemu-snp",label:"Bare metal (SEV-SNP)",children:(0,r.jsxs)(n.p,{children:["A running CoCo-enabled cluster is required for these steps, see the ",(0,r.jsx)(n.a,{href:"/contrast/pr-preview/pr-1285/1.5/getting-started/bare-metal",children:"setup guide"})," on how to set up a bare-metal cluster."]})}),(0,r.jsx)(t,{value:"k3s-qemu-snp-gpu",label:"Bare metal (SEV-SNP, with GPU support)",children:(0,r.jsxs)(n.p,{children:["A running CoCo-enabled cluster is required for these steps, see the ",(0,r.jsx)(n.a,{href:"/contrast/pr-preview/pr-1285/1.5/getting-started/bare-metal",children:"setup guide"})," on how to set up a bare-metal cluster."]})}),(0,r.jsx)(t,{value:"k3s-qemu-tdx",label:"Bare metal (TDX)",children:(0,r.jsxs)(n.p,{children:["A running CoCo-enabled cluster is required for these steps, see the ",(0,r.jsx)(n.a,{href:"/contrast/pr-preview/pr-1285/1.5/getting-started/bare-metal",children:"setup guide"})," on how to set up a bare-metal cluster."]})})]}),"\n",(0,r.jsx)(n.h2,{id:"deploy-the-contrast-runtime",children:"Deploy the Contrast runtime"}),"\n",(0,r.jsxs)(n.p,{children:["Contrast depends on a ",(0,r.jsxs)(n.a,{href:"/contrast/pr-preview/pr-1285/1.5/components/runtime",children:["custom Kubernetes ",(0,r.jsx)(n.code,{children:"RuntimeClass"})," (",(0,r.jsx)(n.code,{children:"contrast-cc"}),")"]}),",\nwhich needs to be installed in the cluster prior to the Coordinator or any confidential workloads.\nThis consists of a ",(0,r.jsx)(n.code,{children:"RuntimeClass"})," resource and a ",(0,r.jsx)(n.code,{children:"DaemonSet"})," that performs installation on worker nodes.\nThis step is only required once for each version of the runtime.\nIt can be shared between Contrast deployments."]}),"\n",(0,r.jsxs)(s,{queryString:"platform",children:[(0,r.jsx)(t,{value:"aks-clh-snp",label:"AKS",default:!0,children:(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-sh",children:"kubectl apply -f https://github.com/edgelesssys/contrast/releases/download/v1.5.1/runtime-aks-clh-snp.yml\n"})})}),(0,r.jsx)(t,{value:"k3s-qemu-snp",label:"Bare metal (SEV-SNP)",children:(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-sh",children:"kubectl apply -f https://github.com/edgelesssys/contrast/releases/download/v1.5.1/runtime-k3s-qemu-snp.yml\n"})})}),(0,r.jsx)(t,{value:"k3s-qemu-snp-gpu",label:"Bare metal (SEV-SNP, with GPU support)",children:(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-sh",children:"kubectl apply -f https://github.com/edgelesssys/contrast/releases/download/v1.5.1/runtime-k3s-qemu-snp-gpu.yml\n"})})}),(0,r.jsx)(t,{value:"k3s-qemu-tdx",label:"Bare metal (TDX)",children:(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-sh",children:"kubectl apply -f https://github.com/edgelesssys/contrast/releases/download/v1.5.1/runtime-k3s-qemu-tdx.yml\n"})})})]}),"\n",(0,r.jsx)(n.h2,{id:"deploy-the-contrast-coordinator",children:"Deploy the Contrast Coordinator"}),"\n",(0,r.jsx)(n.p,{children:"Install the latest Contrast Coordinator release, comprising a single replica deployment and a\nLoadBalancer service, into your cluster."}),"\n",(0,r.jsxs)(s,{queryString:"platform",children:[(0,r.jsx)(t,{value:"aks-clh-snp",label:"AKS",default:!0,children:(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-sh",children:"kubectl apply -f https://github.com/edgelesssys/contrast/releases/download/v1.5.1/coordinator-aks-clh-snp.yml\n"})})}),(0,r.jsx)(t,{value:"k3s-qemu-snp",label:"Bare metal (SEV-SNP)",children:(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-sh",children:"kubectl apply -f https://github.com/edgelesssys/contrast/releases/download/v1.5.1/coordinator-k3s-qemu-snp.yml\n"})})}),(0,r.jsx)(t,{value:"k3s-qemu-snp-gpu",label:"Bare metal (SEV-SNP, with GPU support)",children:(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-sh",children:"kubectl apply -f https://github.com/edgelesssys/contrast/releases/download/v1.5.1/coordinator-k3s-qemu-snp-gpu.yml\n"})})}),(0,r.jsx)(t,{value:"k3s-qemu-tdx",label:"Bare metal (TDX)",children:(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-sh",children:"kubectl apply -f https://github.com/edgelesssys/contrast/releases/download/v1.5.1/coordinator-k3s-qemu-tdx.yml\n"})})})]}),"\n",(0,r.jsx)(n.h2,{id:"prepare-your-kubernetes-resources",children:"Prepare your Kubernetes resources"}),"\n",(0,r.jsx)(n.p,{children:"Your Kubernetes resources need some modifications to run as Confidential Containers.\nThis section guides you through the process and outlines the necessary changes."}),"\n",(0,r.jsx)(n.h3,{id:"security-review",children:"Security review"}),"\n",(0,r.jsxs)(n.p,{children:["Contrast ensures integrity and confidentiality of the applications, but interactions with untrusted systems require the developers' attention.\nReview the ",(0,r.jsx)(n.a,{href:"/contrast/pr-preview/pr-1285/1.5/architecture/security-considerations",children:"security considerations"})," and the ",(0,r.jsx)(n.a,{href:"/contrast/pr-preview/pr-1285/1.5/architecture/certificates",children:"certificates"})," section for writing secure Contrast application."]}),"\n",(0,r.jsx)(n.h3,{id:"runtimeclass",children:"RuntimeClass"}),"\n",(0,r.jsx)(n.p,{children:"Contrast will add annotations to your Kubernetes YAML files. If you want to keep the original files\nunchanged, you can copy the files into a separate local directory.\nYou can also generate files from a Helm chart or from a Kustomization."}),"\n",(0,r.jsxs)(s,{groupId:"yaml-source",children:[(0,r.jsx)(t,{value:"kustomize",label:"kustomize",children:(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-sh",children:"mkdir resources\nkustomize build $MY_RESOURCE_DIR > resources/all.yml\n"})})}),(0,r.jsx)(t,{value:"helm",label:"helm",children:(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-sh",children:"mkdir resources\nhelm template $RELEASE_NAME $CHART_NAME > resources/all.yml\n"})})}),(0,r.jsx)(t,{value:"copy",label:"copy",children:(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-sh",children:"cp -R $MY_RESOURCE_DIR resources/\n"})})})]}),"\n",(0,r.jsxs)(n.p,{children:["To specify that a workload (pod, deployment, etc.) should be deployed as confidential containers,\nadd ",(0,r.jsx)(n.code,{children:"runtimeClassName: contrast-cc"})," to the pod spec (pod definition or template).\nThis is a placeholder name that will be replaced by a versioned ",(0,r.jsx)(n.code,{children:"runtimeClassName"})," when generating policies."]}),"\n",(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-yaml",children:"spec: # v1.PodSpec\n  runtimeClassName: contrast-cc\n"})}),"\n",(0,r.jsx)(n.h3,{id:"pod-resources",children:"Pod resources"}),"\n",(0,r.jsxs)(n.p,{children:["Contrast workloads are deployed as one confidential virtual machine (CVM) per pod.\nIn order to configure the CVM resources correctly, Contrast workloads require a stricter specification of pod resources compared to standard ",(0,r.jsx)(n.a,{href:"https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/",children:"Kubernetes resource management"}),"."]}),"\n",(0,r.jsxs)(n.p,{children:["The total memory available to the CVM is calculated from the sum of the individual containers' memory limits and a static ",(0,r.jsx)(n.code,{children:"RuntimeClass"})," overhead that accounts for services running inside the CVM.\nConsider the following abbreviated example resource definitions:"]}),"\n",(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-yaml",children:'kind: RuntimeClass\nhandler: contrast-cc\noverhead:\n  podFixed:\n    memory: 256Mi\n---\nspec: # v1.PodSpec\n  containers:\n  - name: my-container\n    image: "my-image@sha256:..."\n    resources:\n      limits:\n        memory: 128Mi\n  - name: my-sidecar\n    image: "my-other-image@sha256:..."\n    resources:\n      limits:\n        memory: 64Mi\n'})}),"\n",(0,r.jsx)(n.p,{children:"Contrast launches this pod as a VM with 448MiB of memory: 192MiB for the containers and 256MiB for the Linux kernel, the Kata agent and other base processes."}),"\n",(0,r.jsx)(n.p,{children:"When calculating the VM resource requirements, init containers aren't taken into account.\nIf you have an init container that requires large amounts of memory, you need to adjust the memory limit of one of the main containers in the pod.\nSince memory can't be shared dynamically with the host, each container should have a memory limit that covers its worst-case requirements."}),"\n",(0,r.jsxs)(n.p,{children:["Kubernetes packs a node until the sum of pod ",(0,r.jsx)(n.em,{children:"requests"})," reaches the node's total memory.\nSince a Contrast pod is always going to consume node memory according to the ",(0,r.jsx)(n.em,{children:"limits"}),", the accounting is only correct if the request is equal to the limit.\nThus, once you determined the memory requirements of your application, you should add a resource section to the pod specification with request and limit:"]}),"\n",(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-yaml",children:'spec: # v1.PodSpec\n  containers:\n  - name: my-container\n    image: "my-image@sha256:..."\n    resources:\n      requests:\n        memory: 50Mi\n      limits:\n        memory: 50Mi\n'})}),"\n",(0,r.jsx)(n.admonition,{type:"note",children:(0,r.jsxs)(n.p,{children:["On bare metal platforms, container images are pulled from within the guest CVM and stored in encrypted memory.\nThe CVM mounts a ",(0,r.jsx)(n.code,{children:"tmpfs"})," for the image layers that's capped at 50% of the total VM memory.\nThis tmpfs holds the extracted image layers, so the uncompressed image size needs to be taken into account when setting the container limits.\nRegistry interfaces often show the compressed size of an image, the decompressed image is usually a factor of 2-4x larger if the content is mostly binary.\nFor example, the ",(0,r.jsx)(n.code,{children:"nginx:stable"})," image reports a compressed image size of 67MiB, but storing the uncompressed layers needs about 184MiB of memory.\nAlthough only the extracted layers are stored, and those layers are reused across containers within the same pod, the memory limit should account for both the compressed and the decompressed layer simultaneously.\nAltogether, setting the limit to 10x the compressed image size should be sufficient for small to medium images."]})}),"\n",(0,r.jsx)(n.h3,{id:"handling-tls",children:"Handling TLS"}),"\n",(0,r.jsxs)(n.p,{children:["In the initialization process, the ",(0,r.jsx)(n.code,{children:"contrast-secrets"})," shared volume is populated with X.509 certificates for your workload.\nThese certificates are used by the ",(0,r.jsx)(n.a,{href:"/contrast/pr-preview/pr-1285/1.5/components/service-mesh",children:"Contrast Service Mesh"}),", but can also be used by your application directly.\nThe following tab group explains the setup for both scenarios."]}),"\n",(0,r.jsxs)(s,{groupId:"tls",children:[(0,r.jsxs)(t,{value:"mesh",label:"Drop-in service mesh",children:[(0,r.jsx)(n.p,{children:"Contrast can be configured to handle TLS in a sidecar container.\nThis is useful for workloads that are hard to configure with custom certificates, like Java applications."}),(0,r.jsx)(n.p,{children:"Configuration of the sidecar depends heavily on the application.\nThe following example is for an application with these properties:"}),(0,r.jsxs)(n.ul,{children:["\n",(0,r.jsx)(n.li,{children:"The container has a main application at TCP port 8001, which should be TLS-wrapped and doesn't require client authentication."}),"\n",(0,r.jsx)(n.li,{children:"The container has a metrics endpoint at TCP port 8080, which should be accessible in plain text."}),"\n",(0,r.jsx)(n.li,{children:"All other endpoints require client authentication."}),"\n",(0,r.jsxs)(n.li,{children:["The app connects to a Kubernetes service ",(0,r.jsx)(n.code,{children:"backend.default:4001"}),", which requires client authentication."]}),"\n"]}),(0,r.jsx)(n.p,{children:"Add the following annotations to your workload:"}),(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-yaml",children:'metadata: # apps/v1.Deployment, apps/v1.DaemonSet, ...\n  annotations:\n    contrast.edgeless.systems/servicemesh-ingress: "main#8001#false##metrics#8080#true"\n    contrast.edgeless.systems/servicemesh-egress: "backend#127.0.0.2:4001#backend.default:4001"\n'})}),(0,r.jsxs)(n.p,{children:["During the ",(0,r.jsx)(n.code,{children:"generate"})," step, this configuration will be translated into a Service Mesh sidecar container which handles TLS connections automatically.\nThe only change required to the app itself is to let it connect to ",(0,r.jsx)(n.code,{children:"127.0.0.2:4001"})," to reach the backend service.\nYou can find more detailed documentation in the ",(0,r.jsx)(n.a,{href:"/contrast/pr-preview/pr-1285/1.5/components/service-mesh",children:"Service Mesh chapter"}),"."]})]}),(0,r.jsxs)(t,{value:"go",label:"Go integration",children:[(0,r.jsxs)(n.p,{children:["The mesh certificate contained in ",(0,r.jsx)(n.code,{children:"certChain.pem"})," authenticates this workload, while the mesh CA certificate ",(0,r.jsx)(n.code,{children:"mesh-ca.pem"})," authenticates its peers.\nYour app should turn on client authentication to ensure peers are running as confidential containers, too.\nSee the ",(0,r.jsx)(n.a,{href:"/contrast/pr-preview/pr-1285/1.5/architecture/certificates",children:"Certificate Authority"})," section for detailed information about these certificates."]}),(0,r.jsx)(n.p,{children:"The following example shows how to configure a Golang app, with error handling omitted for clarity."}),(0,r.jsxs)(s,{groupId:"golang-tls-setup",children:[(0,r.jsx)(t,{value:"client",label:"Client",children:(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-go",children:'caCerts := x509.NewCertPool()\ncaCert, _ := os.ReadFile("/contrast/tls-config/mesh-ca.pem")\ncaCerts.AppendCertsFromPEM(caCert)\ncert, _ := tls.LoadX509KeyPair("/contrast/tls-config/certChain.pem", "/contrast/tls-config/key.pem")\ncfg := &tls.Config{\n  Certificates: []tls.Certificate{cert},\n  RootCAs: caCerts,\n}\n'})})}),(0,r.jsx)(t,{value:"server",label:"Server",children:(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-go",children:'caCerts := x509.NewCertPool()\ncaCert, _ := os.ReadFile("/contrast/tls-config/mesh-ca.pem")\ncaCerts.AppendCertsFromPEM(caCert)\ncert, _ := tls.LoadX509KeyPair("/contrast/tls-config/certChain.pem", "/contrast/tls-config/key.pem")\ncfg := &tls.Config{\n  Certificates: []tls.Certificate{cert},\n  ClientAuth: tls.RequireAndVerifyClientCert,\n  ClientCAs: caCerts,\n}\n'})})})]})]})]}),"\n",(0,r.jsx)(n.h3,{id:"using-gpus",children:"Using GPUs"}),"\n",(0,r.jsxs)(n.p,{children:["If the cluster is ",(0,r.jsx)(n.a,{href:"/contrast/pr-preview/pr-1285/1.5/getting-started/bare-metal#preparing-a-cluster-for-gpu-usage",children:"configured for GPU usage"}),", Pods can use GPU devices if needed."]}),"\n",(0,r.jsxs)(n.p,{children:["To do so, a CDI annotation needs to be added, specifying to use the ",(0,r.jsx)(n.code,{children:"pgpu"})," (passthrough GPU) mode. The ",(0,r.jsx)(n.code,{children:"0"})," corresponds to the PCI device index."]}),"\n",(0,r.jsxs)(n.ul,{children:["\n",(0,r.jsxs)(n.li,{children:["For nodes with a single GPU, this value is always ",(0,r.jsx)(n.code,{children:"0"}),"."]}),"\n",(0,r.jsxs)(n.li,{children:["For nodes with multiple GPUs, the value needs to correspond to the device's order as enumerated on the PCI bus. You can identify this order by inspecting the ",(0,r.jsx)(n.code,{children:"/var/run/cdi/nvidia.com-pgpu.yaml"})," file on the specific node."]}),"\n"]}),"\n",(0,r.jsx)(n.p,{children:"This process ensures the correct GPU is allocated to the workload."}),"\n",(0,r.jsxs)(n.p,{children:["As the footprint of a GPU-enabled pod-VM is larger than one of a non-GPU one, the memory of the pod-VM can be adjusted by using the ",(0,r.jsx)(n.code,{children:"io.katacontainers.config.hypervisor.default_memory"})," annotation, which receives the memory the\nVM should receive in MiB. The example below sets it to 16 GB. A reasonable minimum for a GPU pod with a light workload is 8 GB."]}),"\n",(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-yaml",children:'metadata:\n  # ...\n  annotations:\n    # ...\n    cdi.k8s.io/gpu: "nvidia.com/pgpu=0"\n    io.katacontainers.config.hypervisor.default_memory: "16384"\n'})}),"\n",(0,r.jsxs)(n.p,{children:["In addition, the container within the pod that requires GPU access must include a device request.\nThis request specifies the number of GPUs the container should use.\nThe identifiers for the GPUs, obtained during the ",(0,r.jsx)(n.a,{href:"/contrast/pr-preview/pr-1285/1.5/getting-started/bare-metal#preparing-a-cluster-for-gpu-usage",children:"deployment of the NVIDIA GPU Operator"}),", must be included in the request.\nIn the provided example, the container is allocated a single NVIDIA H100 GPU."]}),"\n",(0,r.jsxs)(n.p,{children:["Finally, the environment variable ",(0,r.jsx)(n.code,{children:"NVIDIA_VISIBLE_DEVICES"})," must be set to ",(0,r.jsx)(n.code,{children:"all"})," to grant the container access to GPU utilities provided by the pod-VM. This includes essential tools like CUDA libraries, which are required for running GPU workloads."]}),"\n",(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-yaml",children:'spec:\n  # ...\n  containers:\n  - # ...\n    resources:\n      limits:\n        "nvidia.com/GH100_H100_PCIE": 1\n    env:\n    # ...\n    - name: NVIDIA_VISIBLE_DEVICES\n      value: all\n'})}),"\n",(0,r.jsx)(n.admonition,{type:"note",children:(0,r.jsx)(n.p,{children:"A pod configured to use GPU support may take a few minutes to come up, as the VM creation and boot procedure needs to do more work compared to a non-GPU pod."})}),"\n",(0,r.jsx)(n.h2,{id:"generate-policy-annotations-and-manifest",children:"Generate policy annotations and manifest"}),"\n",(0,r.jsxs)(n.p,{children:["Run the ",(0,r.jsx)(n.code,{children:"generate"})," command to add the necessary components to your deployment files.\nThis will add the Contrast Initializer to every workload with the specified ",(0,r.jsx)(n.code,{children:"contrast-cc"})," runtime class\nand the Contrast Service Mesh to all workloads that have a specified configuration.\nAfter that, it will generate the execution policies and add them as annotations to your deployment files.\nA ",(0,r.jsx)(n.code,{children:"manifest.json"})," with the reference values of your deployment will be created."]}),"\n",(0,r.jsxs)(s,{queryString:"platform",children:[(0,r.jsx)(t,{value:"aks-clh-snp",label:"AKS",default:!0,children:(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-sh",children:"contrast generate --reference-values aks-clh-snp resources/\n"})})}),(0,r.jsxs)(t,{value:"k3s-qemu-snp",label:"Bare metal (SEV-SNP)",children:[(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-sh",children:"contrast generate --reference-values k3s-qemu-snp resources/\n"})}),(0,r.jsx)(n.admonition,{title:"Missing TCB values",type:"note",children:(0,r.jsxs)(n.p,{children:["On bare-metal SEV-SNP, ",(0,r.jsx)(n.code,{children:"contrast generate"})," is unable to fill in the ",(0,r.jsx)(n.code,{children:"MinimumTCB"})," values as they can vary between platforms.\nThey will have to be filled in manually.\nIf you don't know the correct values use ",(0,r.jsx)(n.code,{children:'{"BootloaderVersion":255,"TEEVersion":255,"SNPVersion":255,"MicrocodeVersion":255}'})," and observe the real values in the error messages in the following steps. This should only be done in a secure environment. Note that the values will differ between CPU models."]})})]}),(0,r.jsxs)(t,{value:"k3s-qemu-snp-gpu",label:"Bare metal (SEV-SNP, with GPU support)",children:[(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-sh",children:"contrast generate --reference-values k3s-qemu-snp-gpu resources/\n"})}),(0,r.jsx)(n.admonition,{title:"Missing TCB values",type:"note",children:(0,r.jsxs)(n.p,{children:["On bare-metal SEV-SNP, ",(0,r.jsx)(n.code,{children:"contrast generate"})," is unable to fill in the ",(0,r.jsx)(n.code,{children:"MinimumTCB"})," values as they can vary between platforms.\nThey will have to be filled in manually.\nIf you don't know the correct values use ",(0,r.jsx)(n.code,{children:'{"BootloaderVersion":255,"TEEVersion":255,"SNPVersion":255,"MicrocodeVersion":255}'})," and observe the real values in the error messages in the following steps. This should only be done in a secure environment. Note that the values will differ between CPU models."]})})]}),(0,r.jsxs)(t,{value:"k3s-qemu-tdx",label:"Bare metal (TDX)",children:[(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-sh",children:"contrast generate --reference-values k3s-qemu-tdx resources/\n"})}),(0,r.jsx)(n.admonition,{title:"Missing TCB values",type:"note",children:(0,r.jsxs)(n.p,{children:["On bare-metal TDX, ",(0,r.jsx)(n.code,{children:"contrast generate"})," is unable to fill in the ",(0,r.jsx)(n.code,{children:"MinimumTeeTcbSvn"})," and ",(0,r.jsx)(n.code,{children:"MrSeam"})," TCB values as they can vary between platforms.\nThey will have to be filled in manually.\nIf you don't know the correct values use ",(0,r.jsx)(n.code,{children:"ffffffffffffffffffffffffffffffff"})," and ",(0,r.jsx)(n.code,{children:"000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"})," respectively and observe the real values in the error messages in the following steps. This should only be done in a secure environment."]})})]})]}),"\n",(0,r.jsx)(n.admonition,{type:"warning",children:(0,r.jsxs)(n.p,{children:["Please be aware that runtime policies currently have some blind spots. For example, they can't guarantee the starting order of containers. See the ",(0,r.jsx)(n.a,{href:"/contrast/pr-preview/pr-1285/1.5/features-limitations#runtime-policies",children:"current limitations"})," for more details."]})}),"\n",(0,r.jsxs)(n.p,{children:["Running ",(0,r.jsx)(n.code,{children:"contrast generate"})," for the first time creates some additional files in the working directory:"]}),"\n",(0,r.jsxs)(n.ul,{children:["\n",(0,r.jsxs)(n.li,{children:[(0,r.jsx)(n.code,{children:"seedshare-owner.pem"})," is required for handling the secret seed and recovering the Coordinator (see ",(0,r.jsx)(n.a,{href:"/contrast/pr-preview/pr-1285/1.5/architecture/secrets",children:"Secrets & recovery"}),")."]}),"\n",(0,r.jsxs)(n.li,{children:[(0,r.jsx)(n.code,{children:"workload-owner.pem"})," is required for manifest updates after the initial ",(0,r.jsx)(n.code,{children:"contrast set"}),"."]}),"\n",(0,r.jsxs)(n.li,{children:[(0,r.jsx)(n.code,{children:"rules.rego"})," and ",(0,r.jsx)(n.code,{children:"settings.json"})," are the basis for ",(0,r.jsx)(n.a,{href:"/contrast/pr-preview/pr-1285/1.5/components/policies",children:"runtime policies"}),"."]}),"\n",(0,r.jsxs)(n.li,{children:[(0,r.jsx)(n.code,{children:"layers-cache.json"})," caches container image layer information for your deployments to speed up subsequent runs of ",(0,r.jsx)(n.code,{children:"contrast generate"}),"."]}),"\n"]}),"\n",(0,r.jsx)(n.p,{children:"If you don't want the Contrast Initializer to automatically be added to your\nworkloads, there are two ways you can skip the Initializer injection step,\ndepending on how you want to customize your deployment."}),"\n",(0,r.jsxs)(s,{groupId:"injection",children:[(0,r.jsxs)(t,{value:"flag",label:"Command-line flag",children:[(0,r.jsxs)(n.p,{children:["You can disable the Initializer injection completely by specifying the\n",(0,r.jsx)(n.code,{children:"--skip-initializer"})," flag in the ",(0,r.jsx)(n.code,{children:"generate"})," command."]}),(0,r.jsxs)(s,{queryString:"platform",children:[(0,r.jsx)(t,{value:"aks-clh-snp",label:"AKS",default:!0,children:(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-sh",children:"contrast generate --reference-values aks-clh-snp --skip-initializer resources/\n"})})}),(0,r.jsx)(t,{value:"k3s-qemu-snp",label:"Bare metal (SEV-SNP)",children:(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-sh",children:"contrast generate --reference-values k3s-qemu-snp --skip-initializer resources/\n"})})}),(0,r.jsx)(t,{value:"k3s-qemu-snp-gpu",label:"Bare metal (SEV-SNP, with GPU support)",children:(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-sh",children:"contrast generate --reference-values k3s-qemu-snp-gpu --skip-initializer resources/\n"})})}),(0,r.jsx)(t,{value:"k3s-qemu-tdx",label:"Bare metal (TDX)",children:(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-sh",children:"contrast generate --reference-values k3s-qemu-tdx --skip-initializer resources/\n"})})})]})]}),(0,r.jsxs)(t,{value:"annotation",label:"Per-workload annotation",children:[(0,r.jsxs)(n.p,{children:["If you want to disable the Initializer injection for a specific workload with\nthe ",(0,r.jsx)(n.code,{children:"contrast-cc"})," runtime class, you can do so by adding an annotation to the workload."]}),(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-yaml",children:'metadata: # apps/v1.Deployment, apps/v1.DaemonSet, ...\n  annotations:\n    contrast.edgeless.systems/skip-initializer: "true"\n'})})]})]}),"\n",(0,r.jsxs)(n.p,{children:["When disabling the automatic Initializer injection, you can manually add the\nInitializer as a sidecar container to your workload before generating the\npolicies. Configure the workload to use the certificates written to the\n",(0,r.jsx)(n.code,{children:"contrast-secrets"})," ",(0,r.jsx)(n.code,{children:"volumeMount"}),"."]}),"\n",(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-yaml",children:'# v1.PodSpec\nspec:\n  initContainers:\n    - env:\n        - name: COORDINATOR_HOST\n          value: coordinator\n      image: "ghcr.io/edgelesssys/contrast/initializer:v1.5.1@sha256:6663c11ee05b77870572279d433fe24dc5ef6490392ee29a923243cfc40f2f35"\n      name: contrast-initializer\n      volumeMounts:\n        - mountPath: /contrast\n          name: contrast-secrets\n  volumes:\n    - emptyDir: {}\n      name: contrast-secrets\n'})}),"\n",(0,r.jsx)(n.h2,{id:"apply-the-resources",children:"Apply the resources"}),"\n",(0,r.jsx)(n.p,{children:"Apply the resources to the cluster. Your workloads will block in the initialization phase until a\nmanifest is set at the Coordinator."}),"\n",(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-sh",children:"kubectl apply -f resources/\n"})}),"\n",(0,r.jsx)(n.h2,{id:"connect-to-the-contrast-coordinator",children:"Connect to the Contrast Coordinator"}),"\n",(0,r.jsx)(n.p,{children:"For the next steps, we will need to connect to the Coordinator. The released Coordinator resource\nincludes a LoadBalancer definition we can use."}),"\n",(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-sh",children:"coordinator=$(kubectl get svc coordinator -o=jsonpath='{.status.loadBalancer.ingress[0].ip}')\n"})}),"\n",(0,r.jsxs)(n.admonition,{title:"Port-forwarding of Confidential Containers",type:"info",children:[(0,r.jsxs)(n.p,{children:[(0,r.jsx)(n.code,{children:"kubectl port-forward"})," uses a Container Runtime Interface (CRI) method that isn't supported by the Kata shim.\nIf you can't use a public load balancer, you can deploy a ",(0,r.jsx)(n.a,{href:"https://github.com/edgelesssys/contrast/blob/ddc371b/deployments/emojivoto/portforwarder.yml",children:"port-forwarder"}),".\nThe port-forwarder relays traffic from a CoCo pod and can be accessed via ",(0,r.jsx)(n.code,{children:"kubectl port-forward"}),"."]}),(0,r.jsxs)(n.p,{children:["Upstream tracking issue: ",(0,r.jsx)(n.a,{href:"https://github.com/kata-containers/kata-containers/issues/1693",children:"https://github.com/kata-containers/kata-containers/issues/1693"}),"."]})]}),"\n",(0,r.jsx)(n.h2,{id:"set-the-manifest",children:"Set the manifest"}),"\n",(0,r.jsx)(n.p,{children:"Attest the Coordinator and set the manifest:"}),"\n",(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-sh",children:'contrast set -c "${coordinator}:1313" resources/\n'})}),"\n",(0,r.jsx)(n.p,{children:"This will use the reference values from the manifest file to attest the Coordinator.\nAfter this step, the Coordinator will start issuing TLS certificates to the workloads. The init container\nwill fetch a certificate for the workload and the workload is started."}),"\n",(0,r.jsx)(n.admonition,{type:"warning",children:(0,r.jsxs)(n.p,{children:["On bare metal, the ",(0,r.jsx)(n.a,{href:"/contrast/pr-preview/pr-1285/1.5/components/policies#platform-differences",children:"coordinator policy hash"})," must be overwritten using ",(0,r.jsx)(n.code,{children:"--coordinator-policy-hash"}),"."]})}),"\n",(0,r.jsx)(n.h2,{id:"verify-the-coordinator",children:"Verify the Coordinator"}),"\n",(0,r.jsxs)(n.p,{children:["An end user (data owner) can verify the Contrast deployment using the ",(0,r.jsx)(n.code,{children:"verify"})," command."]}),"\n",(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-sh",children:'contrast verify -c "${coordinator}:1313"\n'})}),"\n",(0,r.jsxs)(n.p,{children:["The CLI will attest the Coordinator using the reference values from the given manifest file. It will then write the\nservice mesh root certificate and the history of manifests into the ",(0,r.jsx)(n.code,{children:"verify/"})," directory. In addition, the policies\nreferenced in the active manifest are also written to the directory. The verification will fail if the active\nmanifest at the Coordinator doesn't match the manifest passed to the CLI."]}),"\n",(0,r.jsx)(n.admonition,{type:"warning",children:(0,r.jsxs)(n.p,{children:["On bare metal, the ",(0,r.jsx)(n.a,{href:"/contrast/pr-preview/pr-1285/1.5/components/policies#platform-differences",children:"coordinator policy hash"})," must be overwritten using ",(0,r.jsx)(n.code,{children:"--coordinator-policy-hash"}),"."]})}),"\n",(0,r.jsx)(n.h2,{id:"communicate-with-workloads",children:"Communicate with workloads"}),"\n",(0,r.jsxs)(n.p,{children:["You can securely connect to the workloads using the Coordinator's ",(0,r.jsx)(n.code,{children:"mesh-ca.pem"})," as a trusted CA certificate.\nFirst, expose the service on a public IP address via a LoadBalancer service:"]}),"\n",(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-sh",children:"kubectl patch svc ${MY_SERVICE} -p '{\"spec\": {\"type\": \"LoadBalancer\"}}'\nkubectl wait --timeout=30s --for=jsonpath='{.status.loadBalancer.ingress}' service/${MY_SERVICE}\nlbip=$(kubectl get svc ${MY_SERVICE} -o=jsonpath='{.status.loadBalancer.ingress[0].ip}')\necho $lbip\n"})}),"\n",(0,r.jsxs)(n.admonition,{title:"Subject alternative names and LoadBalancer IP",type:"info",children:[(0,r.jsx)(n.p,{children:"By default, mesh certificates are issued with a wildcard DNS entry. The web frontend is accessed\nvia load balancer IP in this demo. Tools like curl check the certificate for IP entries in the SAN field.\nValidation fails since the certificate contains no IP entries as a subject alternative name (SAN).\nFor example, attempting to connect with curl and the mesh CA certificate will throw the following error:"}),(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-sh",children:"$ curl --cacert ./verify/mesh-ca.pem \"https://${frontendIP}:443\"\ncurl: (60) SSL: no alternative certificate subject name matches target host name '203.0.113.34'\n"})})]}),"\n",(0,r.jsxs)(n.p,{children:["Using ",(0,r.jsx)(n.code,{children:"openssl"}),", the certificate of the service can be validated with the ",(0,r.jsx)(n.code,{children:"mesh-ca.pem"}),":"]}),"\n",(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-sh",children:"openssl s_client -CAfile verify/mesh-ca.pem -verify_return_error -connect ${frontendIP}:443 < /dev/null\n"})}),"\n",(0,r.jsx)(n.h2,{id:"recover-the-coordinator",children:"Recover the Coordinator"}),"\n",(0,r.jsx)(n.p,{children:"If the Contrast Coordinator restarts, it enters recovery mode and waits for an operator to provide key material.\nFor demonstration purposes, you can simulate this scenario by deleting the Coordinator pod."}),"\n",(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-sh",children:"kubectl delete pod -l app.kubernetes.io/name=coordinator\n"})}),"\n",(0,r.jsxs)(n.p,{children:["Kubernetes schedules a new pod, but that pod doesn't have access to the key material the previous pod held in memory and can't issue certificates for workloads yet.\nYou can confirm this by running ",(0,r.jsx)(n.code,{children:"verify"})," again, or you can restart a workload pod, which should stay in the initialization phase.\nHowever, you can recover the Coordinator using the secret seed and the seed share owner key in your working directory."]}),"\n",(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-sh",children:'contrast recover -c "${coordinator}:1313"\n'})}),"\n",(0,r.jsx)(n.p,{children:"Now that the Coordinator is recovered, all workloads should pass initialization and enter the running state.\nYou can now verify the Coordinator again, which should return the same manifest you set before."}),"\n",(0,r.jsx)(n.admonition,{type:"warning",children:(0,r.jsx)(n.p,{children:"The recovery process invalidates the mesh CA certificate:\nexisting workloads won't be able to communicate with workloads newly spawned.\nAll workloads should be restarted after the recovery succeeded."})}),"\n",(0,r.jsx)(n.admonition,{type:"warning",children:(0,r.jsxs)(n.p,{children:["On bare metal, the ",(0,r.jsx)(n.a,{href:"/contrast/pr-preview/pr-1285/1.5/components/policies#platform-differences",children:"coordinator policy hash"})," must be overwritten using ",(0,r.jsx)(n.code,{children:"--coordinator-policy-hash"}),"."]})})]})}function h(e={}){const{wrapper:n}={...(0,i.R)(),...e.components};return n?(0,r.jsx)(n,{...e,children:(0,r.jsx)(d,{...e})}):d(e)}function u(e,n){throw new Error("Expected "+(n?"component":"object")+" `"+e+"` to be defined: you likely forgot to import, pass, or provide it.")}},28453:(e,n,t)=>{t.d(n,{R:()=>a,x:()=>o});var s=t(96540);const r={},i=s.createContext(r);function a(e){const n=s.useContext(i);return s.useMemo((function(){return"function"==typeof e?e(n):{...n,...e}}),[n,e])}function o(e){let n;return n=e.disableParentContext?"function"==typeof e.components?e.components(r):e.components||r:a(e.components),s.createElement(i.Provider,{value:n},e.children)}}}]);