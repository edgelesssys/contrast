"use strict";(self.webpackChunkcontrast_docs=self.webpackChunkcontrast_docs||[]).push([[2254],{603:(t,e,n)=>{n.r(e),n.d(e,{assets:()=>l,contentTitle:()=>i,default:()=>m,frontMatter:()=>c,metadata:()=>a,toc:()=>u});var r=n(4848),s=n(8453),o=n(4074);const c={},i="Getting started",a={id:"getting-started/index",title:"Getting started",description:"",source:"@site/docs/getting-started/index.md",sourceDirName:"getting-started",slug:"/getting-started/",permalink:"/contrast/pr-preview/pr-605/next/getting-started/",draft:!1,unlisted:!1,editUrl:"https://github.com/edgelesssys/contrast/edit/main/docs/docs/getting-started/index.md",tags:[],version:"current",frontMatter:{},sidebar:"docs",previous:{title:"Features",permalink:"/contrast/pr-preview/pr-605/next/basics/features"},next:{title:"Install",permalink:"/contrast/pr-preview/pr-605/next/getting-started/install"}},l={},u=[];function d(t){const e={h1:"h1",...(0,s.R)(),...t.components};return(0,r.jsxs)(r.Fragment,{children:[(0,r.jsx)(e.h1,{id:"getting-started",children:"Getting started"}),"\n","\n",(0,r.jsx)(o.A,{})]})}function m(t={}){const{wrapper:e}={...(0,s.R)(),...t.components};return e?(0,r.jsx)(e,{...t,children:(0,r.jsx)(d,{...t})}):d(t)}},4074:(t,e,n)=>{n.d(e,{A:()=>C});var r=n(6540),s=n(4164),o=n(5215),c=n(4783),i=n(7639);const a=["zero","one","two","few","many","other"];function l(t){return a.filter((e=>t.includes(e)))}const u={locale:"en",pluralForms:l(["one","other"]),select:t=>1===t?"one":"other"};function d(){const{i18n:{currentLocale:t}}=(0,i.A)();return(0,r.useMemo)((()=>{try{return function(t){const e=new Intl.PluralRules(t);return{locale:t,pluralForms:l(e.resolvedOptions().pluralCategories),select:t=>e.select(t)}}(t)}catch(e){return console.error(`Failed to use Intl.PluralRules for locale "${t}".\nDocusaurus will fallback to the default (English) implementation.\nError: ${e.message}\n`),u}}),[t])}function m(){const t=d();return{selectMessage:(e,n)=>function(t,e,n){const r=t.split("|");if(1===r.length)return r[0];r.length>n.pluralForms.length&&console.error(`For locale=${n.locale}, a maximum of ${n.pluralForms.length} plural forms are expected (${n.pluralForms.join(",")}), but the message contains ${r.length}: ${t}`);const s=n.select(e),o=n.pluralForms.indexOf(s);return r[Math.min(o,r.length-1)]}(n,e,t)}}var p=n(877),f=n(3230),h=n(5225);const g={cardContainer:"cardContainer_fWXF",cardTitle:"cardTitle_rnsV",cardDescription:"cardDescription_PWke"};var x=n(4848);function j(t){let{href:e,children:n}=t;return(0,x.jsx)(c.A,{href:e,className:(0,s.A)("card padding--lg",g.cardContainer),children:n})}function w(t){let{href:e,icon:n,title:r,description:o}=t;return(0,x.jsxs)(j,{href:e,children:[(0,x.jsxs)(h.A,{as:"h2",className:(0,s.A)("text--truncate",g.cardTitle),title:r,children:[n," ",r]}),o&&(0,x.jsx)("p",{className:(0,s.A)("text--truncate",g.cardDescription),title:o,children:o})]})}function v(t){let{item:e}=t;const n=(0,o.Nr)(e),r=function(){const{selectMessage:t}=m();return e=>t(e,(0,f.T)({message:"{count} items",id:"theme.docs.DocCard.categoryDescription.plurals",description:"The default description for a category card in the generated index about how many items this category includes"},{count:e}))}();return n?(0,x.jsx)(w,{href:n,icon:"\ud83d\uddc3\ufe0f",title:e.label,description:e.description??r(e.items.length)}):null}function y(t){let{item:e}=t;const n=(0,p.A)(e.href)?"\ud83d\udcc4\ufe0f":"\ud83d\udd17",r=(0,o.cC)(e.docId??void 0);return(0,x.jsx)(w,{href:e.href,icon:n,title:e.label,description:e.description??r?.description})}function b(t){let{item:e}=t;switch(e.type){case"link":return(0,x.jsx)(y,{item:e});case"category":return(0,x.jsx)(v,{item:e});default:throw new Error(`unknown item type ${JSON.stringify(e)}`)}}function k(t){let{className:e}=t;const n=(0,o.$S)();return(0,x.jsx)(C,{items:n.items,className:e})}function C(t){const{items:e,className:n}=t;if(!e)return(0,x.jsx)(k,{...t});const r=(0,o.d1)(e);return(0,x.jsx)("section",{className:(0,s.A)("row",n),children:r.map(((t,e)=>(0,x.jsx)("article",{className:"col col--6 margin-bottom--lg",children:(0,x.jsx)(b,{item:t})},e)))})}},8453:(t,e,n)=>{n.d(e,{R:()=>c,x:()=>i});var r=n(6540);const s={},o=r.createContext(s);function c(t){const e=r.useContext(o);return r.useMemo((function(){return"function"==typeof t?t(e):{...e,...t}}),[e,t])}function i(t){let e;return e=t.disableParentContext?"function"==typeof t.components?t.components(s):t.components||s:c(t.components),r.createElement(o.Provider,{value:e},t.children)}}}]);