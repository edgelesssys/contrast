"use strict";(self.webpackChunkcontrast_docs=self.webpackChunkcontrast_docs||[]).push([[9013],{7726:(e,n,o)=>{o.r(n),o.d(n,{assets:()=>l,contentTitle:()=>s,default:()=>h,frontMatter:()=>r,metadata:()=>d,toc:()=>c});var t=o(4848),i=o(8453);const r={},s="Troubleshooting",d={id:"troubleshooting",title:"Troubleshooting",description:"This section contains information on how to debug your Contrast deployment.",source:"@site/docs/troubleshooting.md",sourceDirName:".",slug:"/troubleshooting",permalink:"/contrast/pr-preview/pr-632/next/troubleshooting",draft:!1,unlisted:!1,editUrl:"https://github.com/edgelesssys/contrast/edit/main/docs/docs/troubleshooting.md",tags:[],version:"current",frontMatter:{},sidebar:"docs",previous:{title:"Workload deployment",permalink:"/contrast/pr-preview/pr-632/next/deployment"},next:{title:"Components",permalink:"/contrast/pr-preview/pr-632/next/components/"}},l={},c=[{value:"Logging",id:"logging",level:2},{value:"CLI",id:"cli",level:3},{value:"Coordinator and Initializer",id:"coordinator-and-initializer",level:3},{value:"Coordinator debug logging",id:"coordinator-debug-logging",level:4},{value:"Pod fails on startup",id:"pod-fails-on-startup",level:4}];function a(e){const n={code:"code",h1:"h1",h2:"h2",h3:"h3",h4:"h4",li:"li",p:"p",pre:"pre",ul:"ul",...(0,i.R)(),...e.components};return(0,t.jsxs)(t.Fragment,{children:[(0,t.jsx)(n.h1,{id:"troubleshooting",children:"Troubleshooting"}),"\n",(0,t.jsx)(n.p,{children:"This section contains information on how to debug your Contrast deployment."}),"\n",(0,t.jsx)(n.h2,{id:"logging",children:"Logging"}),"\n",(0,t.jsx)(n.p,{children:"Collecting logs can be a good first step to identify problems in your\ndeployment. Both the CLI and the Contrast Coordinator as well as the Initializer\ncan be configured to emit additional logs."}),"\n",(0,t.jsx)(n.h3,{id:"cli",children:"CLI"}),"\n",(0,t.jsxs)(n.p,{children:["The CLI logs can be configured with the ",(0,t.jsx)(n.code,{children:"--log-level"})," command-line flag, which\ncan be set to either ",(0,t.jsx)(n.code,{children:"debug"}),", ",(0,t.jsx)(n.code,{children:"info"}),", ",(0,t.jsx)(n.code,{children:"warn"})," or ",(0,t.jsx)(n.code,{children:"error"}),". The default is ",(0,t.jsx)(n.code,{children:"info"}),".\nSetting this to ",(0,t.jsx)(n.code,{children:"debug"})," can get more fine-grained information as to where the\nproblem lies."]}),"\n",(0,t.jsx)(n.h3,{id:"coordinator-and-initializer",children:"Coordinator and Initializer"}),"\n",(0,t.jsxs)(n.p,{children:["The logs from the Coordinator and the Initializer can be configured via the\nenvironment variables ",(0,t.jsx)(n.code,{children:"CONTRAST_LOG_LEVEL"}),", ",(0,t.jsx)(n.code,{children:"CONTRAST_LOG_FORMAT"})," and\n",(0,t.jsx)(n.code,{children:"CONTRAST_LOG_SUBSYSTEMS"}),"."]}),"\n",(0,t.jsxs)(n.ul,{children:["\n",(0,t.jsxs)(n.li,{children:[(0,t.jsx)(n.code,{children:"CONTRAST_LOG_LEVEL"})," can be set to one of either ",(0,t.jsx)(n.code,{children:"debug"}),", ",(0,t.jsx)(n.code,{children:"info"}),", ",(0,t.jsx)(n.code,{children:"warn"}),", or\n",(0,t.jsx)(n.code,{children:"error"}),", similar to the CLI (defaults to ",(0,t.jsx)(n.code,{children:"info"}),")."]}),"\n",(0,t.jsxs)(n.li,{children:[(0,t.jsx)(n.code,{children:"CONTRAST_LOG_FORMAT"})," can be set to ",(0,t.jsx)(n.code,{children:"text"})," or ",(0,t.jsx)(n.code,{children:"json"}),", determining the output\nformat (defaults to ",(0,t.jsx)(n.code,{children:"text"}),")."]}),"\n",(0,t.jsxs)(n.li,{children:[(0,t.jsx)(n.code,{children:"CONTRAST_LOG_SUBSYSTEMS"})," is a comma-seperated list of subsystems that should\nbe enabled for logging, which are disabled by default. Subsystems include:\n",(0,t.jsx)(n.code,{children:"snp-issuer"}),", ",(0,t.jsx)(n.code,{children:"kds-getter"}),", and ",(0,t.jsx)(n.code,{children:"snp-validator"}),". To enable all subsystems, use\n",(0,t.jsx)(n.code,{children:"*"})," as the value for this environment variable."]}),"\n"]}),"\n",(0,t.jsxs)(n.p,{children:["Warnings and error messages from subsystems get printed regardless of whether\nthe subsystem is listed in the ",(0,t.jsx)(n.code,{children:"CONTRAST_LOG_SUBSYSTEMS"})," environment variable."]}),"\n",(0,t.jsx)(n.h4,{id:"coordinator-debug-logging",children:"Coordinator debug logging"}),"\n",(0,t.jsx)(n.p,{children:"To configure debug logging with all subsystems for your Coordinator, add the\nfollowing variables to your container definition."}),"\n",(0,t.jsx)(n.pre,{children:(0,t.jsx)(n.code,{className:"language-yaml",children:'spec: # v1.PodSpec\n  containers:\n    image: "ghcr.io/edgelesssys/contrast/coordinator:latest"\n    name: coordinator\n    env:\n    - name: CONTRAST_LOG_LEVEL\n      value: debug\n    - name: CONTRAST_LOG_SUBSYSTEMS\n      value: "*"\n    # ...\n'})}),"\n",(0,t.jsxs)(n.p,{children:["To access the logs generated by the Coordinator, you can use ",(0,t.jsx)(n.code,{children:"kubectl"})," with the\nfollowing command."]}),"\n",(0,t.jsx)(n.pre,{children:(0,t.jsx)(n.code,{className:"language-sh",children:"kubectl logs <coordinator-pod-name>\n"})}),"\n",(0,t.jsx)(n.h4,{id:"pod-fails-on-startup",children:"Pod fails on startup"}),"\n",(0,t.jsxs)(n.p,{children:["If the Coordinator or a workload pod fails to even start, it can be helpful to\nlook at the events of the pod during the startup process using the ",(0,t.jsx)(n.code,{children:"describe"}),"\ncommand."]}),"\n",(0,t.jsx)(n.pre,{children:(0,t.jsx)(n.code,{className:"language-sh",children:"kubectl describe pod <pod-name>\n"})}),"\n",(0,t.jsx)(n.p,{children:"Example output:"}),"\n",(0,t.jsx)(n.pre,{children:(0,t.jsx)(n.code,{children:'Events:\n  Type     Reason   Age    From     Message\n  ----     ------   ----   ----     -------\n  ...\n  Warning  Failed   20s    kubelet  Error: failed to create containerd task: failed to create shim task: "CreateContainerRequest is blocked by policy: ...\n'})}),"\n",(0,t.jsxs)(n.p,{children:["In this example, the container creation was blocked by a policy. This suggests\nthat a policy hasn't been updated to accommodate recent changes to the\nconfiguration. Make sure to run ",(0,t.jsx)(n.code,{children:"contrast generate"})," when altering your\ndeployment."]})]})}function h(e={}){const{wrapper:n}={...(0,i.R)(),...e.components};return n?(0,t.jsx)(n,{...e,children:(0,t.jsx)(a,{...e})}):a(e)}},8453:(e,n,o)=>{o.d(n,{R:()=>s,x:()=>d});var t=o(6540);const i={},r=t.createContext(i);function s(e){const n=t.useContext(r);return t.useMemo((function(){return"function"==typeof e?e(n):{...n,...e}}),[n,e])}function d(e){let n;return n=e.disableParentContext?"function"==typeof e.components?e.components(i):e.components||i:s(e.components),t.createElement(r.Provider,{value:n},e.children)}}}]);