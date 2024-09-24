"use strict";(self.webpackChunkcontrast_docs=self.webpackChunkcontrast_docs||[]).push([[4206],{5115:(e,t,n)=>{n.r(t),n.d(t,{assets:()=>l,contentTitle:()=>c,default:()=>m,frontMatter:()=>i,metadata:()=>a,toc:()=>d});var r=n(4848),s=n(8453),o=n(4074);const i={},c="Getting started",a={id:"getting-started/index",title:"Getting started",description:"",source:"@site/versioned_docs/version-0.6/getting-started/index.md",sourceDirName:"getting-started",slug:"/getting-started/",permalink:"/contrast/pr-preview/pr-887/0.6/getting-started/",draft:!1,unlisted:!1,editUrl:"https://github.com/edgelesssys/contrast/edit/main/docs/versioned_docs/version-0.6/getting-started/index.md",tags:[],version:"0.6",frontMatter:{},sidebar:"docs",previous:{title:"Features",permalink:"/contrast/pr-preview/pr-887/0.6/basics/features"},next:{title:"Install",permalink:"/contrast/pr-preview/pr-887/0.6/getting-started/install"}},l={},d=[];function u(e){const t={h1:"h1",header:"header",...(0,s.R)(),...e.components};return(0,r.jsxs)(r.Fragment,{children:[(0,r.jsx)(t.header,{children:(0,r.jsx)(t.h1,{id:"getting-started",children:"Getting started"})}),"\n","\n",(0,r.jsx)(o.A,{})]})}function m(e={}){const{wrapper:t}={...(0,s.R)(),...e.components};return t?(0,r.jsx)(t,{...e,children:(0,r.jsx)(u,{...e})}):u(e)}},4074:(e,t,n)=>{n.d(t,{A:()=>C});var r=n(6540),s=n(4164),o=n(5357),i=n(4783),c=n(7639);const a=["zero","one","two","few","many","other"];function l(e){return a.filter((t=>e.includes(t)))}const d={locale:"en",pluralForms:l(["one","other"]),select:e=>1===e?"one":"other"};function u(){const{i18n:{currentLocale:e}}=(0,c.A)();return(0,r.useMemo)((()=>{try{return function(e){const t=new Intl.PluralRules(e);return{locale:e,pluralForms:l(t.resolvedOptions().pluralCategories),select:e=>t.select(e)}}(e)}catch(t){return console.error(`Failed to use Intl.PluralRules for locale "${e}".\nDocusaurus will fallback to the default (English) implementation.\nError: ${t.message}\n`),d}}),[e])}function m(){const e=u();return{selectMessage:(t,n)=>function(e,t,n){const r=e.split("|");if(1===r.length)return r[0];r.length>n.pluralForms.length&&console.error(`For locale=${n.locale}, a maximum of ${n.pluralForms.length} plural forms are expected (${n.pluralForms.join(",")}), but the message contains ${r.length}: ${e}`);const s=n.select(t),o=n.pluralForms.indexOf(s);return r[Math.min(o,r.length-1)]}(n,t,e)}}var p=n(877),h=n(3230),f=n(5225);const g={cardContainer:"cardContainer_fWXF",cardTitle:"cardTitle_rnsV",cardDescription:"cardDescription_PWke"};var x=n(4848);function j(e){let{href:t,children:n}=e;return(0,x.jsx)(i.A,{href:t,className:(0,s.A)("card padding--lg",g.cardContainer),children:n})}function v(e){let{href:t,icon:n,title:r,description:o}=e;return(0,x.jsxs)(j,{href:t,children:[(0,x.jsxs)(f.A,{as:"h2",className:(0,s.A)("text--truncate",g.cardTitle),title:r,children:[n," ",r]}),o&&(0,x.jsx)("p",{className:(0,s.A)("text--truncate",g.cardDescription),title:o,children:o})]})}function w(e){let{item:t}=e;const n=(0,o.Nr)(t),r=function(){const{selectMessage:e}=m();return t=>e(t,(0,h.T)({message:"1 item|{count} items",id:"theme.docs.DocCard.categoryDescription.plurals",description:"The default description for a category card in the generated index about how many items this category includes"},{count:t}))}();return n?(0,x.jsx)(v,{href:n,icon:"\ud83d\uddc3\ufe0f",title:t.label,description:t.description??r(t.items.length)}):null}function y(e){let{item:t}=e;const n=(0,p.A)(t.href)?"\ud83d\udcc4\ufe0f":"\ud83d\udd17",r=(0,o.cC)(t.docId??void 0);return(0,x.jsx)(v,{href:t.href,icon:n,title:t.label,description:t.description??r?.description})}function b(e){let{item:t}=e;switch(t.type){case"link":return(0,x.jsx)(y,{item:t});case"category":return(0,x.jsx)(w,{item:t});default:throw new Error(`unknown item type ${JSON.stringify(t)}`)}}function k(e){let{className:t}=e;const n=(0,o.$S)();return(0,x.jsx)(C,{items:n.items,className:t})}function C(e){const{items:t,className:n}=e;if(!t)return(0,x.jsx)(k,{...e});const r=(0,o.d1)(t);return(0,x.jsx)("section",{className:(0,s.A)("row",n),children:r.map(((e,t)=>(0,x.jsx)("article",{className:"col col--6 margin-bottom--lg",children:(0,x.jsx)(b,{item:e})},t)))})}},8453:(e,t,n)=>{n.d(t,{R:()=>i,x:()=>c});var r=n(6540);const s={},o=r.createContext(s);function i(e){const t=r.useContext(o);return r.useMemo((function(){return"function"==typeof e?e(t):{...t,...e}}),[t,e])}function c(e){let t;return t=e.disableParentContext?"function"==typeof e.components?e.components(s):e.components||s:i(e.components),r.createElement(o.Provider,{value:t},e.children)}}}]);