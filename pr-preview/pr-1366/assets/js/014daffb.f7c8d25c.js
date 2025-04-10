"use strict";(self.webpackChunkcontrast_docs=self.webpackChunkcontrast_docs||[]).push([[1955],{28453:(e,t,n)=>{n.d(t,{R:()=>o,x:()=>a});var s=n(96540);const r={},i=s.createContext(r);function o(e){const t=s.useContext(i);return s.useMemo((function(){return"function"==typeof e?e(t):{...t,...e}}),[t,e])}function a(e){let t;return t=e.disableParentContext?"function"==typeof e.components?e.components(r):e.components||r:o(e.components),s.createElement(i.Provider,{value:t},e.children)}},86472:(e,t,n)=>{n.r(t),n.d(t,{assets:()=>c,contentTitle:()=>a,default:()=>l,frontMatter:()=>o,metadata:()=>s,toc:()=>d});const s=JSON.parse('{"id":"architecture/secrets","title":"Secrets & recovery","description":"When the Coordinator is configured with the initial manifest, it generates a random secret seed.","source":"@site/docs/architecture/secrets.md","sourceDirName":"architecture","slug":"/architecture/secrets","permalink":"/contrast/pr-preview/pr-1366/next/architecture/secrets","draft":false,"unlisted":false,"editUrl":"https://github.com/edgelesssys/contrast/edit/main/docs/docs/architecture/secrets.md","tags":[],"version":"current","frontMatter":{},"sidebar":"docs","previous":{"title":"Attestation","permalink":"/contrast/pr-preview/pr-1366/next/architecture/attestation"},"next":{"title":"Certificate authority","permalink":"/contrast/pr-preview/pr-1366/next/architecture/certificates"}}');var r=n(74848),i=n(28453);const o={},a="Secrets & recovery",c={},d=[{value:"Persistence",id:"persistence",level:2},{value:"Recovery",id:"recovery",level:2},{value:"Workload Secrets",id:"workload-secrets",level:2},{value:"Secure persistence",id:"secure-persistence",level:3},{value:"Usage <code>cryptsetup</code> subcommand",id:"usage-cryptsetup-subcommand",level:4}];function h(e){const t={admonition:"admonition",code:"code",h1:"h1",h2:"h2",h3:"h3",h4:"h4",header:"header",li:"li",p:"p",pre:"pre",ul:"ul",...(0,i.R)(),...e.components};return(0,r.jsxs)(r.Fragment,{children:[(0,r.jsx)(t.header,{children:(0,r.jsx)(t.h1,{id:"secrets--recovery",children:"Secrets & recovery"})}),"\n",(0,r.jsx)(t.p,{children:"When the Coordinator is configured with the initial manifest, it generates a random secret seed.\nFrom this seed, it uses an HKDF to derive the CA root key and a signing key for the manifest history.\nThis derivation is deterministic, so the seed can be used to bring any Coordinator to this Coordinator's state."}),"\n",(0,r.jsxs)(t.p,{children:["The secret seed is returned to the user on the first call to ",(0,r.jsx)(t.code,{children:"contrast set"}),", encrypted with the user's public seed share owner key.\nIf no seed share owner key is provided, a key is generated and stored in the working directory."]}),"\n",(0,r.jsxs)(t.admonition,{type:"danger",children:[(0,r.jsx)(t.p,{children:"The secret seed and the seed share owner key are highly sensitive."}),(0,r.jsxs)(t.ul,{children:["\n",(0,r.jsx)(t.li,{children:"If either of them leak, the Contrast deployment should be considered compromised."}),"\n",(0,r.jsx)(t.li,{children:"If the secret seed is lost, data encrypted with Contrast secrets can't be recovered."}),"\n",(0,r.jsx)(t.li,{children:"If the seed share owner key is lost, the Coordinator can't be recovered and needs to be redeployed with a new manifest."}),"\n"]})]}),"\n",(0,r.jsx)(t.h2,{id:"persistence",children:"Persistence"}),"\n",(0,r.jsxs)(t.p,{children:["The Coordinator runs as a ",(0,r.jsx)(t.code,{children:"StatefulSet"})," with a dynamically provisioned persistent volume.\nThis volume stores the manifest history and the associated runtime policies.\nThe manifest isn't considered sensitive information, because it needs to be passed to the untrusted infrastructure in order to start workloads.\nHowever, the Coordinator must ensure its integrity and that the persisted data corresponds to the manifests set by authorized users.\nThus, the manifest is stored in plain text, but is signed with a private key derived from the Coordinator's secret seed."]}),"\n",(0,r.jsx)(t.h2,{id:"recovery",children:"Recovery"}),"\n",(0,r.jsxs)(t.p,{children:["When a Coordinator starts up, it doesn't have access to the signing secret and can thus not verify the integrity of the persisted manifests.\nIt needs to be provided with the secret seed, from which it can derive the signing key that verifies the signatures.\nThis procedure is called recovery and is initiated by the seed share owner.\nThe CLI decrypts the secret seed using the private seed share owner key, verifies the Coordinator and sends the seed through the ",(0,r.jsx)(t.code,{children:"Recover"})," method.\nThe Coordinator authenticates the seed share owner, recovers its key material, and verifies the manifest history signature."]}),"\n",(0,r.jsx)(t.h2,{id:"workload-secrets",children:"Workload Secrets"}),"\n",(0,r.jsxs)(t.p,{children:["The Coordinator provides each workload a secret seed during attestation.\nThis secret can be used by the workload to derive additional secrets for example to encrypt persistent data.\nLike the workload certificates, it's written to the ",(0,r.jsx)(t.code,{children:"secrets/workload-secret-seed"})," path under the shared Kubernetes volume ",(0,r.jsx)(t.code,{children:"contrast-secrets"}),"."]}),"\n",(0,r.jsx)(t.admonition,{type:"warning",children:(0,r.jsx)(t.p,{children:"The seed share owner can decrypt data encrypted with secrets derived from the workload secret, because they can themselves derive the workload secret.\nIf the data owner fully trusts the seed share owner (when they're the same entity, for example), they can securely use the workload secrets."})}),"\n",(0,r.jsx)(t.h3,{id:"secure-persistence",children:"Secure persistence"}),"\n",(0,r.jsxs)(t.p,{children:["Remember that persistent volumes from the cloud provider are untrusted.\nApplications can set up trusted storage on top of an untrusted block device using the ",(0,r.jsx)(t.code,{children:"contrast.edgeless.systems/secure-pv"})," annotation.\nThis annotation enables ",(0,r.jsx)(t.code,{children:"contrast generate"})," to configure the Initializer to set up a LUKS-encrypted volume at the specified device and mount it to a specified volume.\nThe LUKS encryption utilizes the workload secret introduced above.\nConfigure any workload resource with the following annotation:"]}),"\n",(0,r.jsx)(t.pre,{children:(0,r.jsx)(t.code,{className:"language-yaml",children:'metadata:\n  annotations:\n    contrast.edgeless.systems/secure-pv: "device-name:mount-name"\n'})}),"\n",(0,r.jsxs)(t.p,{children:["This requires an existing block device named ",(0,r.jsx)(t.code,{children:"device-name"})," which is configured as a volume on the resource.\nThe volume ",(0,r.jsx)(t.code,{children:"mount-name"})," has to be of type ",(0,r.jsx)(t.code,{children:"EmptyDir"})," and will be created if not present.\nThe resulting Initializer will mount both the device and configured volume and set up the encrypted storage.\nWorkload containers can then use the volume as a secure storage location:"]}),"\n",(0,r.jsx)(t.pre,{children:(0,r.jsx)(t.code,{className:"language-yaml",children:'apiVersion: apps/v1\nkind: StatefulSet\nmetadata:\n  annotations:\n    contrast.edgeless.systems/secure-pv: "device:secure"\n  name: my-statefulset\nspec:\n  template:\n    spec:\n      containers:\n        - name: my-container\n          image: "my-image@sha256:..."\n          volumeMounts:\n            - mountPath: /secure\n              mountPropagation: HostToContainer\n              name: secure\n      volumes:\n        - name: device\n          persistentVolumeClaim:\n            claimName: my-pvc\n      runtimeClassName: contrast-cc\n'})}),"\n",(0,r.jsxs)(t.h4,{id:"usage-cryptsetup-subcommand",children:["Usage ",(0,r.jsx)(t.code,{children:"cryptsetup"})," subcommand"]}),"\n",(0,r.jsxs)(t.p,{children:["Alternatively, the ",(0,r.jsx)(t.code,{children:"cryptsetup"})," subcommand of the Initializer can be used to manually set up encrypted storage.\nThe ",(0,r.jsx)(t.code,{children:"cryptsetup"})," subcommand takes two arguments ",(0,r.jsx)(t.code,{children:"cryptsetup -d [device-path] -m [mount-point]"}),", to set up a LUKS-encrypted volume at ",(0,r.jsx)(t.code,{children:"device-path"})," and mount that volume at ",(0,r.jsx)(t.code,{children:"mount-point"}),"."]}),"\n",(0,r.jsx)(t.p,{children:"The following, slightly abbreviated resource outlines how this could be realized:"}),"\n",(0,r.jsx)(t.admonition,{type:"warning",children:(0,r.jsx)(t.p,{children:"This configuration snippet is intended to be educational and needs to be refined and adapted to your production environment.\nUsing it as-is may result in data corruption or data loss."})}),"\n",(0,r.jsx)(t.pre,{children:(0,r.jsx)(t.code,{className:"language-yaml",children:'apiVersion: apps/v1\nkind: StatefulSet\nmetadata:\n  name: volume-tester\nspec:\n  template:\n    spec:\n      containers:\n        - name: main\n          image: my.registry/my-image@sha256:0123... # <-- Original application requiring encrypted disk.\n          volumeMounts:\n            - mountPath: /state\n              mountPropagation: HostToContainer\n              name: share\n      initContainers:\n        - args:\n            - cryptsetup # <-- cryptsetup subcommand provided as args to the initializer binary.\n            - "--device-path"\n            - /dev/csi0\n            - "--mount-point"\n            - /state\n          image: "ghcr.io/edgelesssys/contrast/initializer:latest"\n          name: encrypted-volume-initializer\n          resources:\n            limits:\n              memory: 100Mi\n            requests:\n              memory: 100Mi\n          restartPolicy: Always\n          securityContext:\n            privileged: true # <-- This is necessary for mounting devices.\n          startupProbe:\n            exec:\n              command:\n                - /bin/test\n                - "-f"\n                - /done\n            failureThreshold: 20\n            periodSeconds: 5\n          volumeDevices:\n            - devicePath: /dev/csi0\n              name: state\n          volumeMounts:\n            - mountPath: /state\n              mountPropagation: Bidirectional\n              name: share\n      volumes:\n        - name: share\n          emptyDir: {}\n      runtimeClassName: contrast-cc\n  volumeClaimTemplates:\n    - apiVersion: v1\n      kind: PersistentVolumeClaim\n      metadata:\n        name: state\n      spec:\n        accessModes:\n          - ReadWriteOnce\n        resources:\n          requests:\n            storage: 1Gi\n        volumeMode: Block # <-- The requested volume needs to be a raw block device.\n'})})]})}function l(e={}){const{wrapper:t}={...(0,i.R)(),...e.components};return t?(0,r.jsx)(t,{...e,children:(0,r.jsx)(h,{...e})}):h(e)}}}]);