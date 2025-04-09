"use strict";(self.webpackChunkcontrast_docs=self.webpackChunkcontrast_docs||[]).push([[3],{30797:(e,t,n)=>{n.r(t),n.d(t,{assets:()=>c,contentTitle:()=>o,default:()=>l,frontMatter:()=>a,metadata:()=>r,toc:()=>d});const r=JSON.parse('{"id":"howto/registry-authentication","title":"Registry authentication","description":"This guide shows how to set up registry credentials for Contrast.","source":"@site/docs/howto/registry-authentication.md","sourceDirName":"howto","slug":"/howto/registry-authentication","permalink":"/contrast/pr-preview/pr-1285/next/howto/registry-authentication","draft":false,"unlisted":false,"editUrl":"https://github.com/edgelesssys/contrast/edit/main/docs/docs/howto/registry-authentication.md","tags":[],"version":"current","frontMatter":{},"sidebar":"docs","previous":{"title":"SNP Attestation","permalink":"/contrast/pr-preview/pr-1285/next/architecture/snp"},"next":{"title":"Planned features and limitations","permalink":"/contrast/pr-preview/pr-1285/next/features-limitations"}}');var s=n(74848),i=n(28453);const a={},o="Registry authentication",c={},d=[{value:"Contrast CLI",id:"contrast-cli",level:2},{value:"AKS",id:"aks",level:2},{value:"Bare metal",id:"bare-metal",level:2}];function h(e){const t={a:"a",code:"code",h1:"h1",h2:"h2",header:"header",p:"p",pre:"pre",...(0,i.R)(),...e.components};return(0,s.jsxs)(s.Fragment,{children:[(0,s.jsx)(t.header,{children:(0,s.jsx)(t.h1,{id:"registry-authentication",children:"Registry authentication"})}),"\n",(0,s.jsx)(t.p,{children:"This guide shows how to set up registry credentials for Contrast."}),"\n",(0,s.jsx)(t.h2,{id:"contrast-cli",children:"Contrast CLI"}),"\n",(0,s.jsxs)(t.p,{children:["The Contrast CLI, specifically the ",(0,s.jsx)(t.code,{children:"contrast generate"})," subcommand, needs access to the registry to derive policies for the referenced container images.\nThe CLI authenticates to the registry using the ",(0,s.jsx)(t.a,{href:"https://crates.io/crates/docker_credential",children:(0,s.jsx)(t.code,{children:"docker_credential"})})," crate.\nThis crate searches some default locations for a registry authentication file, so it should find credentials created by ",(0,s.jsx)(t.code,{children:"docker login"})," or ",(0,s.jsx)(t.code,{children:"podman login"}),"."]}),"\n",(0,s.jsxs)(t.p,{children:["The only authentication method that's currently supported is ",(0,s.jsx)(t.code,{children:"Basic"})," HTTP authentication with user name and password (or personal access token).\nIdentity token flows, such as the default mechanism of plain ",(0,s.jsx)(t.code,{children:"docker login"}),", don't work.\nBasic authentication can be forced with ",(0,s.jsx)(t.code,{children:"docker login -u $REGISTRYUSER"}),"."]}),"\n",(0,s.jsxs)(t.p,{children:["You can override the credentials file used by Contrast by setting the environment variable ",(0,s.jsx)(t.code,{children:"DOCKER_CONFIG"}),". This is useful for creating a credential file from scratch, as shown in the following script:"]}),"\n",(0,s.jsx)(t.pre,{children:(0,s.jsx)(t.code,{className:"language-sh",children:'#!/bin/sh\n\nregistry="<put registry here>"\nuser="<put user id here>"\npassword="<put client secret here>"\n\nexport DOCKER_CONFIG=$(mktemp)\n\ncat >"${DOCKER_CONFIG}" <<EOF\n{\n        "auths": {\n                "$registry": {\n                        "auth": "$(printf "%s:%s" "$user" "$password" | base64 -w0)"\n                }\n        }\n}\nEOF\n\ncontrast generate "$@"\n'})}),"\n",(0,s.jsx)(t.h2,{id:"aks",children:"AKS"}),"\n",(0,s.jsxs)(t.p,{children:["On AKS, images are pulled on the worker nodes using credentials available to Kubernetes and containerd.\nFollow the ",(0,s.jsx)(t.a,{href:"https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/",children:"official instructions"})," to set up registry authentication with image pull secrets."]}),"\n",(0,s.jsx)(t.h2,{id:"bare-metal",children:"Bare metal"}),"\n",(0,s.jsx)(t.p,{children:"On bare metal, images are pulled within the confidential guest, which doesn't receive credentials from the host yet.\nYou can work around this by mirroring the required images to a private registry that's only exposed to the cluster.\nSuch a registry needs to have a valid TLS certificate that's trusted in the web PKI (issued by Let's Encrypt, for example)."})]})}function l(e={}){const{wrapper:t}={...(0,i.R)(),...e.components};return t?(0,s.jsx)(t,{...e,children:(0,s.jsx)(h,{...e})}):h(e)}},28453:(e,t,n)=>{n.d(t,{R:()=>a,x:()=>o});var r=n(96540);const s={},i=r.createContext(s);function a(e){const t=r.useContext(i);return r.useMemo((function(){return"function"==typeof e?e(t):{...t,...e}}),[t,e])}function o(e){let t;return t=e.disableParentContext?"function"==typeof e.components?e.components(s):e.components||s:a(e.components),r.createElement(i.Provider,{value:t},e.children)}}}]);