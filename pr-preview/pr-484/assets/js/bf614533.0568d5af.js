"use strict";(self.webpackChunkcontrast_docs=self.webpackChunkcontrast_docs||[]).push([[495],{8822:(e,t,n)=>{n.r(t),n.d(t,{assets:()=>a,contentTitle:()=>i,default:()=>m,frontMatter:()=>c,metadata:()=>l,toc:()=>u});var r=n(4848),s=n(8453),o=n(4074);const c={},i="Examples",l={id:"examples/index",title:"Examples",description:"",source:"@site/docs/examples/index.md",sourceDirName:"examples",slug:"/examples/",permalink:"/contrast/pr-preview/pr-484/next/examples/",draft:!1,unlisted:!1,editUrl:"https://github.com/edgelesssys/contrast/edit/main/docs/docs/examples/index.md",tags:[],version:"current",frontMatter:{},sidebar:"docs",previous:{title:"Cluster setup",permalink:"/contrast/pr-preview/pr-484/next/getting-started/cluster-setup"},next:{title:"Confidential emoji voting",permalink:"/contrast/pr-preview/pr-484/next/examples/emojivoto"}},a={},u=[];function d(e){const t={h1:"h1",...(0,s.R)(),...e.components};return(0,r.jsxs)(r.Fragment,{children:[(0,r.jsx)(t.h1,{id:"examples",children:"Examples"}),"\n","\n",(0,r.jsx)(o.A,{})]})}function m(e={}){const{wrapper:t}={...(0,s.R)(),...e.components};return t?(0,r.jsx)(t,{...e,children:(0,r.jsx)(d,{...e})}):d(e)}},4074:(e,t,n)=>{n.d(t,{A:()=>k});var r=n(6540),s=n(4164),o=n(5215),c=n(4783),i=n(7639);const l=["zero","one","two","few","many","other"];function a(e){return l.filter((t=>e.includes(t)))}const u={locale:"en",pluralForms:a(["one","other"]),select:e=>1===e?"one":"other"};function d(){const{i18n:{currentLocale:e}}=(0,i.A)();return(0,r.useMemo)((()=>{try{return function(e){const t=new Intl.PluralRules(e);return{locale:e,pluralForms:a(t.resolvedOptions().pluralCategories),select:e=>t.select(e)}}(e)}catch(t){return console.error(`Failed to use Intl.PluralRules for locale "${e}".\nDocusaurus will fallback to the default (English) implementation.\nError: ${t.message}\n`),u}}),[e])}function m(){const e=d();return{selectMessage:(t,n)=>function(e,t,n){const r=e.split("|");if(1===r.length)return r[0];r.length>n.pluralForms.length&&console.error(`For locale=${n.locale}, a maximum of ${n.pluralForms.length} plural forms are expected (${n.pluralForms.join(",")}), but the message contains ${r.length}: ${e}`);const s=n.select(t),o=n.pluralForms.indexOf(s);return r[Math.min(o,r.length-1)]}(n,t,e)}}var p=n(877),f=n(3230),h=n(5225);const x={cardContainer:"cardContainer_fWXF",cardTitle:"cardTitle_rnsV",cardDescription:"cardDescription_PWke"};var g=n(4848);function j(e){let{href:t,children:n}=e;return(0,g.jsx)(c.A,{href:t,className:(0,s.A)("card padding--lg",x.cardContainer),children:n})}function v(e){let{href:t,icon:n,title:r,description:o}=e;return(0,g.jsxs)(j,{href:t,children:[(0,g.jsxs)(h.A,{as:"h2",className:(0,s.A)("text--truncate",x.cardTitle),title:r,children:[n," ",r]}),o&&(0,g.jsx)("p",{className:(0,s.A)("text--truncate",x.cardDescription),title:o,children:o})]})}function w(e){let{item:t}=e;const n=(0,o.Nr)(t),r=function(){const{selectMessage:e}=m();return t=>e(t,(0,f.T)({message:"{count} items",id:"theme.docs.DocCard.categoryDescription.plurals",description:"The default description for a category card in the generated index about how many items this category includes"},{count:t}))}();return n?(0,g.jsx)(v,{href:n,icon:"\ud83d\uddc3\ufe0f",title:t.label,description:t.description??r(t.items.length)}):null}function y(e){let{item:t}=e;const n=(0,p.A)(t.href)?"\ud83d\udcc4\ufe0f":"\ud83d\udd17",r=(0,o.cC)(t.docId??void 0);return(0,g.jsx)(v,{href:t.href,icon:n,title:t.label,description:t.description??r?.description})}function C(e){let{item:t}=e;switch(t.type){case"link":return(0,g.jsx)(y,{item:t});case"category":return(0,g.jsx)(w,{item:t});default:throw new Error(`unknown item type ${JSON.stringify(t)}`)}}function b(e){let{className:t}=e;const n=(0,o.$S)();return(0,g.jsx)(k,{items:n.items,className:t})}function k(e){const{items:t,className:n}=e;if(!t)return(0,g.jsx)(b,{...e});const r=(0,o.d1)(t);return(0,g.jsx)("section",{className:(0,s.A)("row",n),children:r.map(((e,t)=>(0,g.jsx)("article",{className:"col col--6 margin-bottom--lg",children:(0,g.jsx)(C,{item:e})},t)))})}},8453:(e,t,n)=>{n.d(t,{R:()=>c,x:()=>i});var r=n(6540);const s={},o=r.createContext(s);function c(e){const t=r.useContext(o);return r.useMemo((function(){return"function"==typeof e?e(t):{...t,...e}}),[t,e])}function i(e){let t;return t=e.disableParentContext?"function"==typeof e.components?e.components(s):e.components||s:c(e.components),r.createElement(o.Provider,{value:t},e.children)}}}]);