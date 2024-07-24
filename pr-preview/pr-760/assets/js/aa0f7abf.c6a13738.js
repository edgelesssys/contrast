"use strict";(self.webpackChunkcontrast_docs=self.webpackChunkcontrast_docs||[]).push([[8295],{1907:(e,t,r)=>{r.r(t),r.d(t,{assets:()=>l,contentTitle:()=>i,default:()=>m,frontMatter:()=>o,metadata:()=>a,toc:()=>u});var n=r(4848),c=r(8453),s=r(4074);const o={},i="Architecture",a={id:"architecture/index",title:"Architecture",description:"",source:"@site/versioned_docs/version-0.7/architecture/index.md",sourceDirName:"architecture",slug:"/architecture/",permalink:"/contrast/pr-preview/pr-760/0.7/architecture/",draft:!1,unlisted:!1,editUrl:"https://github.com/edgelesssys/contrast/edit/main/docs/versioned_docs/version-0.7/architecture/index.md",tags:[],version:"0.7",frontMatter:{},sidebar:"docs",previous:{title:"Service mesh",permalink:"/contrast/pr-preview/pr-760/0.7/components/service-mesh"},next:{title:"Attestation",permalink:"/contrast/pr-preview/pr-760/0.7/architecture/attestation"}},l={},u=[];function d(e){const t={h1:"h1",...(0,c.R)(),...e.components};return(0,n.jsxs)(n.Fragment,{children:[(0,n.jsx)(t.h1,{id:"architecture",children:"Architecture"}),"\n","\n",(0,n.jsx)(s.A,{})]})}function m(e={}){const{wrapper:t}={...(0,c.R)(),...e.components};return t?(0,n.jsx)(t,{...e,children:(0,n.jsx)(d,{...e})}):d(e)}},4074:(e,t,r)=>{r.d(t,{A:()=>k});var n=r(6540),c=r(4164),s=r(5215),o=r(4783),i=r(7639);const a=["zero","one","two","few","many","other"];function l(e){return a.filter((t=>e.includes(t)))}const u={locale:"en",pluralForms:l(["one","other"]),select:e=>1===e?"one":"other"};function d(){const{i18n:{currentLocale:e}}=(0,i.A)();return(0,n.useMemo)((()=>{try{return function(e){const t=new Intl.PluralRules(e);return{locale:e,pluralForms:l(t.resolvedOptions().pluralCategories),select:e=>t.select(e)}}(e)}catch(t){return console.error(`Failed to use Intl.PluralRules for locale "${e}".\nDocusaurus will fallback to the default (English) implementation.\nError: ${t.message}\n`),u}}),[e])}function m(){const e=d();return{selectMessage:(t,r)=>function(e,t,r){const n=e.split("|");if(1===n.length)return n[0];n.length>r.pluralForms.length&&console.error(`For locale=${r.locale}, a maximum of ${r.pluralForms.length} plural forms are expected (${r.pluralForms.join(",")}), but the message contains ${n.length}: ${e}`);const c=r.select(t),s=r.pluralForms.indexOf(c);return n[Math.min(s,n.length-1)]}(r,t,e)}}var h=r(877),p=r(3230),f=r(5225);const x={cardContainer:"cardContainer_fWXF",cardTitle:"cardTitle_rnsV",cardDescription:"cardDescription_PWke"};var g=r(4848);function v(e){let{href:t,children:r}=e;return(0,g.jsx)(o.A,{href:t,className:(0,c.A)("card padding--lg",x.cardContainer),children:r})}function j(e){let{href:t,icon:r,title:n,description:s}=e;return(0,g.jsxs)(v,{href:t,children:[(0,g.jsxs)(f.A,{as:"h2",className:(0,c.A)("text--truncate",x.cardTitle),title:n,children:[r," ",n]}),s&&(0,g.jsx)("p",{className:(0,c.A)("text--truncate",x.cardDescription),title:s,children:s})]})}function w(e){let{item:t}=e;const r=(0,s.Nr)(t),n=function(){const{selectMessage:e}=m();return t=>e(t,(0,p.T)({message:"{count} items",id:"theme.docs.DocCard.categoryDescription.plurals",description:"The default description for a category card in the generated index about how many items this category includes"},{count:t}))}();return r?(0,g.jsx)(j,{href:r,icon:"\ud83d\uddc3\ufe0f",title:t.label,description:t.description??n(t.items.length)}):null}function A(e){let{item:t}=e;const r=(0,h.A)(t.href)?"\ud83d\udcc4\ufe0f":"\ud83d\udd17",n=(0,s.cC)(t.docId??void 0);return(0,g.jsx)(j,{href:t.href,icon:r,title:t.label,description:t.description??n?.description})}function y(e){let{item:t}=e;switch(t.type){case"link":return(0,g.jsx)(A,{item:t});case"category":return(0,g.jsx)(w,{item:t});default:throw new Error(`unknown item type ${JSON.stringify(t)}`)}}function b(e){let{className:t}=e;const r=(0,s.$S)();return(0,g.jsx)(k,{items:r.items,className:t})}function k(e){const{items:t,className:r}=e;if(!t)return(0,g.jsx)(b,{...e});const n=(0,s.d1)(t);return(0,g.jsx)("section",{className:(0,c.A)("row",r),children:n.map(((e,t)=>(0,g.jsx)("article",{className:"col col--6 margin-bottom--lg",children:(0,g.jsx)(y,{item:e})},t)))})}},8453:(e,t,r)=>{r.d(t,{R:()=>o,x:()=>i});var n=r(6540);const c={},s=n.createContext(c);function o(e){const t=n.useContext(s);return n.useMemo((function(){return"function"==typeof e?e(t):{...t,...e}}),[t,e])}function i(e){let t;return t=e.disableParentContext?"function"==typeof e.components?e.components(c):e.components||c:o(e.components),n.createElement(s.Provider,{value:t},e.children)}}}]);