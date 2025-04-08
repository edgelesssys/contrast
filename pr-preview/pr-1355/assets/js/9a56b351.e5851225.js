"use strict";(self.webpackChunkcontrast_docs=self.webpackChunkcontrast_docs||[]).push([[8863],{71800:(e,t,n)=>{n.r(t),n.d(t,{assets:()=>d,contentTitle:()=>a,default:()=>l,frontMatter:()=>o,metadata:()=>s,toc:()=>c});const s=JSON.parse('{"id":"architecture/secrets","title":"Secrets & recovery","description":"When the Coordinator is configured with the initial manifest, it generates a random secret seed.","source":"@site/versioned_docs/version-1.5/architecture/secrets.md","sourceDirName":"architecture","slug":"/architecture/secrets","permalink":"/contrast/pr-preview/pr-1355/1.5/architecture/secrets","draft":false,"unlisted":false,"editUrl":"https://github.com/edgelesssys/contrast/edit/main/docs/versioned_docs/version-1.5/architecture/secrets.md","tags":[],"version":"1.5","frontMatter":{},"sidebar":"docs","previous":{"title":"Attestation","permalink":"/contrast/pr-preview/pr-1355/1.5/architecture/attestation"},"next":{"title":"Certificate authority","permalink":"/contrast/pr-preview/pr-1355/1.5/architecture/certificates"}}');var r=n(74848),i=n(28453);const o={},a="Secrets & recovery",d={},c=[{value:"Persistence",id:"persistence",level:2},{value:"Recovery",id:"recovery",level:2},{value:"Workload Secrets",id:"workload-secrets",level:2},{value:"Secure persistence",id:"secure-persistence",level:3},{value:"Usage <code>cryptsetup</code> subcommand",id:"usage-cryptsetup-subcommand",level:4}];function h(e){const t={admonition:"admonition",code:"code",h1:"h1",h2:"h2",h3:"h3",h4:"h4",header:"header",li:"li",p:"p",pre:"pre",ul:"ul",...(0,i.R)(),...e.components};return(0,r.jsxs)(r.Fragment,{children:[(0,r.jsx)(t.header,{children:(0,r.jsx)(t.h1,{id:"secrets--recovery",children:"Secrets & recovery"})}),"\n",(0,r.jsx)(t.p,{children:"When the Coordinator is configured with the initial manifest, it generates a random secret seed.\nFrom this seed, it uses an HKDF to derive the CA root key and a signing key for the manifest history.\nThis derivation is deterministic, so the seed can be used to bring any Coordinator to this Coordinator's state."}),"\n",(0,r.jsxs)(t.p,{children:["The secret seed is returned to the user on the first call to ",(0,r.jsx)(t.code,{children:"contrast set"}),", encrypted with the user's public seed share owner key.\nIf no seed share owner key is provided, a key is generated and stored in the working directory."]}),"\n",(0,r.jsxs)(t.admonition,{type:"danger",children:[(0,r.jsx)(t.p,{children:"The secret seed and the seed share owner key are highly sensitive."}),(0,r.jsxs)(t.ul,{children:["\n",(0,r.jsx)(t.li,{children:"If either of them leak, the Contrast deployment should be considered compromised."}),"\n",(0,r.jsx)(t.li,{children:"If the secret seed is lost, data encrypted with Contrast secrets can't be recovered."}),"\n",(0,r.jsx)(t.li,{children:"If the seed share owner key is lost, the Coordinator can't be recovered and needs to be redeployed with a new manifest."}),"\n"]})]}),"\n",(0,r.jsx)(t.h2,{id:"persistence",children:"Persistence"}),"\n",(0,r.jsxs)(t.p,{children:["The Coordinator runs as a ",(0,r.jsx)(t.code,{children:"StatefulSet"})," with a dynamically provisioned persistent volume.\nThis volume stores the manifest history and the associated runtime policies.\nThe manifest isn't considered sensitive information, because it needs to be passed to the untrusted infrastructure in order to start workloads.\nHowever, the Coordinator must ensure its integrity and that the persisted data corresponds to the manifests set by authorized users.\nThus, the manifest is stored in plain text, but is signed with a private key derived from the Coordinator's secret seed."]}),"\n",(0,r.jsx)(t.h2,{id:"recovery",children:"Recovery"}),"\n",(0,r.jsxs)(t.p,{children:["When a Coordinator starts up, it doesn't have access to the signing secret and can thus not verify the integrity of the persisted manifests.\nIt needs to be provided with the secret seed, from which it can derive the signing key that verifies the signatures.\nThis procedure is called recovery and is initiated by the seed share owner.\nThe CLI decrypts the secret seed using the private seed share owner key, verifies the Coordinator and sends the seed through the ",(0,r.jsx)(t.code,{children:"Recover"})," method.\nThe Coordinator authenticates the seed share owner, recovers its key material, and verifies the manifest history signature."]}),"\n",(0,r.jsx)(t.h2,{id:"workload-secrets",children:"Workload Secrets"}),"\n",(0,r.jsxs)(t.p,{children:["The Coordinator provides each workload a secret seed during attestation.\nThis secret can be used by the workload to derive additional secrets for example to encrypt persistent data.\nLike the workload certificates, it's written to the ",(0,r.jsx)(t.code,{children:"secrets/workload-secret-seed"})," path under the shared Kubernetes volume ",(0,r.jsx)(t.code,{children:"contrast-secrets"}),"."]}),"\n",(0,r.jsx)(t.admonition,{type:"warning",children:(0,r.jsx)(t.p,{children:"The seed share owner can decrypt data encrypted with secrets derived from the workload secret, because they can themselves derive the workload secret.\nIf the data owner fully trusts the seed share owner (when they're the same entity, for example), they can securely use the workload secrets."})}),"\n",(0,r.jsx)(t.h3,{id:"secure-persistence",children:"Secure persistence"}),"\n",(0,r.jsxs)(t.p,{children:["Remember that persistent volumes from the cloud provider are untrusted.\nUsing the built-in ",(0,r.jsx)(t.code,{children:"cryptsetup"})," subcommand of the initializer, applications can set up trusted storage on top of untrusted block devices based on the workload secret.\nFunctionally the initializer will act as a sidecar container which serves to set up a secure mount inside an ",(0,r.jsx)(t.code,{children:"emptyDir"})," mount shared with the main container."]}),"\n",(0,r.jsxs)(t.h4,{id:"usage-cryptsetup-subcommand",children:["Usage ",(0,r.jsx)(t.code,{children:"cryptsetup"})," subcommand"]}),"\n",(0,r.jsxs)(t.p,{children:["The ",(0,r.jsx)(t.code,{children:"cryptsetup"})," subcommand takes two arguments ",(0,r.jsx)(t.code,{children:"cryptsetup -d [device-path] -m [mount-point]"}),", to set up a LUKS-encrypted volume at ",(0,r.jsx)(t.code,{children:"device-path"})," and mount that volume at ",(0,r.jsx)(t.code,{children:"mount-point"}),"."]}),"\n",(0,r.jsx)(t.p,{children:"The following, slightly abbreviated resource outlines how this could be realized:"}),"\n",(0,r.jsx)(t.admonition,{type:"warning",children:(0,r.jsx)(t.p,{children:"This configuration snippet is intended to be educational and needs to be refined and adapted to your production environment.\nUsing it as-is may result in data corruption or data loss."})}),"\n",(0,r.jsx)(t.pre,{children:(0,r.jsx)(t.code,{className:"language-yaml",children:'apiVersion: apps/v1\nkind: StatefulSet\nmetadata:\n  name: volume-tester\nspec:\n  template:\n    spec:\n      containers:\n        - name: main\n          image: my.registry/my-image@sha256:0123... # <-- Original application requiring encrypted disk.\n          volumeMounts:\n            - mountPath: /state\n              mountPropagation: HostToContainer\n              name: share\n      initContainers:\n        - args:\n            - cryptsetup # <-- cryptsetup subcommand provided as args to the initializer binary.\n            - "--device-path"\n            - /dev/csi0\n            - "--mount-point"\n            - /state\n          image: "ghcr.io/edgelesssys/contrast/initializer:v1.5.1@sha256:6663c11ee05b77870572279d433fe24dc5ef6490392ee29a923243cfc40f2f35"\n          name: encrypted-volume-initializer\n          resources:\n            limits:\n              memory: 100Mi\n            requests:\n              memory: 100Mi\n          restartPolicy: Always\n          securityContext:\n            privileged: true # <-- This is necessary for mounting devices.\n          startupProbe:\n            exec:\n              command:\n                - /bin/test\n                - "-f"\n                - /done\n            failureThreshold: 20\n            periodSeconds: 5\n          volumeDevices:\n            - devicePath: /dev/csi0\n              name: state\n          volumeMounts:\n            - mountPath: /state\n              mountPropagation: Bidirectional\n              name: share\n      volumes:\n        - name: share\n          emptyDir: {}\n      runtimeClassName: contrast-cc\n  volumeClaimTemplates:\n    - apiVersion: v1\n      kind: PersistentVolumeClaim\n      metadata:\n        name: state\n      spec:\n        accessModes:\n          - ReadWriteOnce\n        resources:\n          requests:\n            storage: 1Gi\n        volumeMode: Block # <-- The requested volume needs to be a raw block device.\n'})})]})}function l(e={}){const{wrapper:t}={...(0,i.R)(),...e.components};return t?(0,r.jsx)(t,{...e,children:(0,r.jsx)(h,{...e})}):h(e)}},28453:(e,t,n)=>{n.d(t,{R:()=>o,x:()=>a});var s=n(96540);const r={},i=s.createContext(r);function o(e){const t=s.useContext(i);return s.useMemo((function(){return"function"==typeof e?e(t):{...t,...e}}),[t,e])}function a(e){let t;return t=e.disableParentContext?"function"==typeof e.components?e.components(r):e.components||r:o(e.components),s.createElement(i.Provider,{value:t},e.children)}}}]);