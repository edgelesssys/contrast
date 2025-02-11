"use strict";(self.webpackChunkcontrast_docs=self.webpackChunkcontrast_docs||[]).push([[6463],{56430:(e,n,t)=>{t.r(n),t.d(n,{assets:()=>c,contentTitle:()=>r,default:()=>h,frontMatter:()=>a,metadata:()=>o,toc:()=>l});const o=JSON.parse('{"id":"troubleshooting","title":"Troubleshooting","description":"This section contains information on how to debug your Contrast deployment.","source":"@site/versioned_docs/version-1.2/troubleshooting.md","sourceDirName":".","slug":"/troubleshooting","permalink":"/contrast/pr-preview/pr-1222/1.2/troubleshooting","draft":false,"unlisted":false,"editUrl":"https://github.com/edgelesssys/contrast/edit/main/docs/versioned_docs/version-1.2/troubleshooting.md","tags":[],"version":"1.2","frontMatter":{},"sidebar":"docs","previous":{"title":"Workload deployment","permalink":"/contrast/pr-preview/pr-1222/1.2/deployment"},"next":{"title":"Overview","permalink":"/contrast/pr-preview/pr-1222/1.2/components/overview"}}');var i=t(74848),s=t(28453);const a={},r="Troubleshooting",c={},l=[{value:"Logging",id:"logging",level:2},{value:"CLI",id:"cli",level:3},{value:"Coordinator and Initializer",id:"coordinator-and-initializer",level:3},{value:"Pod fails to start",id:"pod-fails-to-start",level:2},{value:"Regenerating the policies",id:"regenerating-the-policies",level:3},{value:"Pin container images",id:"pin-container-images",level:3},{value:"Validate Contrast components match",id:"validate-contrast-components-match",level:3}];function d(e){const n={admonition:"admonition",code:"code",h1:"h1",h2:"h2",h3:"h3",header:"header",li:"li",p:"p",pre:"pre",ul:"ul",...(0,s.R)(),...e.components};return(0,i.jsxs)(i.Fragment,{children:[(0,i.jsx)(n.header,{children:(0,i.jsx)(n.h1,{id:"troubleshooting",children:"Troubleshooting"})}),"\n",(0,i.jsx)(n.p,{children:"This section contains information on how to debug your Contrast deployment."}),"\n",(0,i.jsx)(n.h2,{id:"logging",children:"Logging"}),"\n",(0,i.jsx)(n.p,{children:"Collecting logs can be a good first step to identify problems in your\ndeployment. Both the CLI and the Contrast Coordinator as well as the Initializer\ncan be configured to emit additional logs."}),"\n",(0,i.jsx)(n.h3,{id:"cli",children:"CLI"}),"\n",(0,i.jsxs)(n.p,{children:["The CLI logs can be configured with the ",(0,i.jsx)(n.code,{children:"--log-level"})," command-line flag, which\ncan be set to either ",(0,i.jsx)(n.code,{children:"debug"}),", ",(0,i.jsx)(n.code,{children:"info"}),", ",(0,i.jsx)(n.code,{children:"warn"})," or ",(0,i.jsx)(n.code,{children:"error"}),". The default is ",(0,i.jsx)(n.code,{children:"info"}),".\nSetting this to ",(0,i.jsx)(n.code,{children:"debug"})," can get more fine-grained information as to where the\nproblem lies."]}),"\n",(0,i.jsx)(n.h3,{id:"coordinator-and-initializer",children:"Coordinator and Initializer"}),"\n",(0,i.jsxs)(n.p,{children:["The logs from the Coordinator and the Initializer can be configured via the\nenvironment variables ",(0,i.jsx)(n.code,{children:"CONTRAST_LOG_LEVEL"}),", ",(0,i.jsx)(n.code,{children:"CONTRAST_LOG_FORMAT"})," and\n",(0,i.jsx)(n.code,{children:"CONTRAST_LOG_SUBSYSTEMS"}),"."]}),"\n",(0,i.jsxs)(n.ul,{children:["\n",(0,i.jsxs)(n.li,{children:[(0,i.jsx)(n.code,{children:"CONTRAST_LOG_LEVEL"})," can be set to one of either ",(0,i.jsx)(n.code,{children:"debug"}),", ",(0,i.jsx)(n.code,{children:"info"}),", ",(0,i.jsx)(n.code,{children:"warn"}),", or\n",(0,i.jsx)(n.code,{children:"error"}),", similar to the CLI (defaults to ",(0,i.jsx)(n.code,{children:"info"}),")."]}),"\n",(0,i.jsxs)(n.li,{children:[(0,i.jsx)(n.code,{children:"CONTRAST_LOG_FORMAT"})," can be set to ",(0,i.jsx)(n.code,{children:"text"})," or ",(0,i.jsx)(n.code,{children:"json"}),", determining the output\nformat (defaults to ",(0,i.jsx)(n.code,{children:"text"}),")."]}),"\n",(0,i.jsxs)(n.li,{children:[(0,i.jsx)(n.code,{children:"CONTRAST_LOG_SUBSYSTEMS"})," is a comma-separated list of subsystems that should\nbe enabled for logging, which are disabled by default. Subsystems include:\n",(0,i.jsx)(n.code,{children:"kds-getter"}),", ",(0,i.jsx)(n.code,{children:"issuer"})," and ",(0,i.jsx)(n.code,{children:"validator"}),".\nTo enable all subsystems, use ",(0,i.jsx)(n.code,{children:"*"})," as the value for this environment variable.\nWarnings and error messages from subsystems get printed regardless of whether\nthe subsystem is listed in the ",(0,i.jsx)(n.code,{children:"CONTRAST_LOG_SUBSYSTEMS"})," environment variable."]}),"\n"]}),"\n",(0,i.jsx)(n.p,{children:"To configure debug logging with all subsystems for your Coordinator, add the\nfollowing variables to your container definition."}),"\n",(0,i.jsx)(n.pre,{children:(0,i.jsx)(n.code,{className:"language-yaml",children:'spec: # v1.PodSpec\n  containers:\n    image: "ghcr.io/edgelesssys/contrast/coordinator:v1.2.1@sha256:cd5776968b645672edfcc4da1784800ac43fb9386ae25910ab24a7010860410b"\n    name: coordinator\n    env:\n    - name: CONTRAST_LOG_LEVEL\n      value: debug\n    - name: CONTRAST_LOG_SUBSYSTEMS\n      value: "*"\n    # ...\n'})}),"\n",(0,i.jsx)(n.admonition,{type:"info",children:(0,i.jsxs)(n.p,{children:["While the Contrast Coordinator has a policy that allows certain configurations,\nthe Initializer and service mesh don't. When changing environment variables of other\nparts than the Coordinator, ensure to rerun ",(0,i.jsx)(n.code,{children:"contrast generate"})," to update the policy."]})}),"\n",(0,i.jsxs)(n.p,{children:["To access the logs generated by the Coordinator, you can use ",(0,i.jsx)(n.code,{children:"kubectl"})," with the\nfollowing command:"]}),"\n",(0,i.jsx)(n.pre,{children:(0,i.jsx)(n.code,{className:"language-sh",children:"kubectl logs <coordinator-pod-name>\n"})}),"\n",(0,i.jsx)(n.h2,{id:"pod-fails-to-start",children:"Pod fails to start"}),"\n",(0,i.jsxs)(n.p,{children:["If the Coordinator or a workload pod fails to even start, it can be helpful to\nlook at the events of the pod during the startup process using the ",(0,i.jsx)(n.code,{children:"describe"}),"\ncommand."]}),"\n",(0,i.jsx)(n.pre,{children:(0,i.jsx)(n.code,{className:"language-sh",children:"kubectl -n <namespace> events --for pod/<coordinator-pod-name>\n"})}),"\n",(0,i.jsx)(n.p,{children:"Example output:"}),"\n",(0,i.jsx)(n.pre,{children:(0,i.jsx)(n.code,{children:'LAST SEEN  TYPE     REASON  OBJECT             MESSAGE\n32m        Warning  Failed  Pod/coordinator-0  kubelet  Error: failed to create containerd task: failed to create shim task: "CreateContainerRequest is blocked by policy: ...\n'})}),"\n",(0,i.jsx)(n.p,{children:"A common error, as in this example, is that the container creation was blocked by the\npolicy. Potential reasons are a modification of the deployment YAML without updating\nthe policies afterward, or a version mismatch between Contrast components."}),"\n",(0,i.jsx)(n.h3,{id:"regenerating-the-policies",children:"Regenerating the policies"}),"\n",(0,i.jsx)(n.p,{children:"To ensure there isn't a mismatch between Kubernetes resource YAML and the annotated\npolicies, rerun"}),"\n",(0,i.jsx)(n.pre,{children:(0,i.jsx)(n.code,{className:"language-sh",children:"contrast generate\n"})}),"\n",(0,i.jsx)(n.p,{children:"on your deployment. If any of the policy annotations change, re-deploy with the updated policies."}),"\n",(0,i.jsx)(n.h3,{id:"pin-container-images",children:"Pin container images"}),"\n",(0,i.jsx)(n.p,{children:"When generating the policies, Contrast will download the images specified in your deployment\nYAML and include their cryptographic identity. If the image tag is moved to another\ncontainer image after the policy has been generated, the image downloaded at deploy time\nwill differ from the one at generation time, and the policy enforcement won't allow the\ncontainer to be started in the pod VM."}),"\n",(0,i.jsxs)(n.p,{children:["To ensure the correct image is always used, pin the container image to a fixed ",(0,i.jsx)(n.code,{children:"sha256"}),":"]}),"\n",(0,i.jsx)(n.pre,{children:(0,i.jsx)(n.code,{className:"language-yaml",children:"image: ubuntu:22.04@sha256:19478ce7fc2ffbce89df29fea5725a8d12e57de52eb9ea570890dc5852aac1ac\n"})}),"\n",(0,i.jsxs)(n.p,{children:["This way, the same image will still be pulled when the container tag (",(0,i.jsx)(n.code,{children:"22.04"}),") is moved\nto another image."]}),"\n",(0,i.jsx)(n.h3,{id:"validate-contrast-components-match",children:"Validate Contrast components match"}),"\n",(0,i.jsx)(n.p,{children:"A version mismatch between Contrast components can cause policy validation or attestation\nto fail. Each Contrast runtime is identifiable based on its (shortened) measurement value\nused to name the runtime class version."}),"\n",(0,i.jsx)(n.p,{children:"First, analyze which runtime class is currently installed in your cluster by running"}),"\n",(0,i.jsx)(n.pre,{children:(0,i.jsx)(n.code,{className:"language-sh",children:"kubectl get runtimeclasses\n"})}),"\n",(0,i.jsx)(n.p,{children:"This should give you output similar to the following one."}),"\n",(0,i.jsx)(n.pre,{children:(0,i.jsx)(n.code,{className:"language-sh",children:"NAME                                           HANDLER                                        AGE\ncontrast-cc-aks-clh-snp-7173acb5               contrast-cc-aks-clh-snp-7173acb5               23h\nkata-cc-isolation                              kata-cc                                        45d\n"})}),"\n",(0,i.jsx)(n.p,{children:"The output shows that there are four Contrast runtime classes installed (as well as the runtime class provided\nby the AKS CoCo preview, which isn't used by Contrast)."}),"\n",(0,i.jsx)(n.p,{children:"Next, check if the pod that won't start has the correct runtime class configured, and the\nCoordinator uses the exact same runtime:"}),"\n",(0,i.jsx)(n.pre,{children:(0,i.jsx)(n.code,{className:"language-sh",children:"kubectl -n <namespace> get -o=jsonpath='{.spec.runtimeClassName}' pod/<pod-name>\nkubectl -n <namespace> get -o=jsonpath='{.spec.runtimeClassName}' pod/<coordinator-pod-name>\n"})}),"\n",(0,i.jsx)(n.p,{children:"The output should list the runtime class the pod is using:"}),"\n",(0,i.jsx)(n.pre,{children:(0,i.jsx)(n.code,{className:"language-sh",children:"contrast-cc-aks-clh-snp-7173acb5\n"})}),"\n",(0,i.jsxs)(n.p,{children:["Version information about the currently used CLI can be obtained via the ",(0,i.jsx)(n.code,{children:"version"})," flag:"]}),"\n",(0,i.jsx)(n.pre,{children:(0,i.jsx)(n.code,{className:"language-sh",children:"contrast --version\n"})}),"\n",(0,i.jsx)(n.pre,{children:(0,i.jsx)(n.code,{className:"language-sh",children:"contrast version v0.X.0\n\n    runtime handler:      contrast-cc-aks-clh-snp-7173acb5\n    launch digest:        beee79ca916b9e5dc59602788cbfb097721cde34943e1583a3918f21011a71c47f371f68e883f5e474a6d4053d931a35\n    genpolicy version:    3.2.0.azl1.genpolicy0\n    image versions:       ghcr.io/edgelesssys/contrast/coordinator@sha256:...\n                          ghcr.io/edgelesssys/contrast/initializer@sha256:...\n"})})]})}function h(e={}){const{wrapper:n}={...(0,s.R)(),...e.components};return n?(0,i.jsx)(n,{...e,children:(0,i.jsx)(d,{...e})}):d(e)}},28453:(e,n,t)=>{t.d(n,{R:()=>a,x:()=>r});var o=t(96540);const i={},s=o.createContext(i);function a(e){const n=o.useContext(s);return o.useMemo((function(){return"function"==typeof e?e(n):{...n,...e}}),[n,e])}function r(e){let n;return n=e.disableParentContext?"function"==typeof e.components?e.components(i):e.components||i:a(e.components),o.createElement(s.Provider,{value:n},e.children)}}}]);