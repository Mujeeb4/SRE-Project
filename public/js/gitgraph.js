(window.webpackJsonp=window.webpackJsonp||[]).push([[0],[,,,,,,,,,,function(t,e,n){(t.exports=n(11)(!1)).push([t.i,"/* This is a customized version of https://github.com/bluef/gitgraph.js/blob/master/gitgraph.css\n   Changes include the removal of `body` and `em` styles */\n#git-graph-container, #rel-container {float:left;}\n#rel-container {max-width:30%; overflow-x:auto;}\n#git-graph-container {overflow-x:auto; width:100%}\n#git-graph-container li {list-style-type:none;height:20px;line-height:20px; white-space:nowrap;}\n#git-graph-container li .node-relation {font-family:'Bitstream Vera Sans Mono', 'Courier', monospace;}\n#git-graph-container li .author {color:#666666;}\n#git-graph-container li .time {color:#999999;font-size:80%}\n#git-graph-container li a {color:#000000;}\n#git-graph-container li a:hover {text-decoration:underline;}\n#git-graph-container li a em {color:#BB0000;border-bottom:1px dotted #BBBBBB;text-decoration:none;font-style:normal;}\n#rev-container {width:100%}\n#rev-list {margin:0;padding:0 5px 0 5px;min-width:95%}\n#graph-raw-list {margin:0px;}\n",""])},function(t,e,n){"use strict";t.exports=function(t){var e=[];return e.toString=function(){return this.map((function(e){var n=function(t,e){var n=t[1]||"",i=t[3];if(!i)return n;if(e&&"function"==typeof btoa){var r=(a=i,c=btoa(unescape(encodeURIComponent(JSON.stringify(a)))),s="sourceMappingURL=data:application/json;charset=utf-8;base64,".concat(c),"/*# ".concat(s," */")),o=i.sources.map((function(t){return"/*# sourceURL=".concat(i.sourceRoot).concat(t," */")}));return[n].concat(o).concat([r]).join("\n")}var a,c,s;return[n].join("\n")}(e,t);return e[2]?"@media ".concat(e[2],"{").concat(n,"}"):n})).join("")},e.i=function(t,n){"string"==typeof t&&(t=[[null,t,""]]);for(var i={},r=0;r<this.length;r++){var o=this[r][0];null!=o&&(i[o]=!0)}for(var a=0;a<t.length;a++){var c=t[a];null!=c[0]&&i[c[0]]||(n&&!c[2]?c[2]=n:n&&(c[2]="(".concat(c[2],") and (").concat(n,")")),e.push(c))}},e}},function(t,e,n){"use strict";var i,r={},o=function(){return void 0===i&&(i=Boolean(window&&document&&document.all&&!window.atob)),i},a=function(){var t={};return function(e){if(void 0===t[e]){var n=document.querySelector(e);if(window.HTMLIFrameElement&&n instanceof window.HTMLIFrameElement)try{n=n.contentDocument.head}catch(t){n=null}t[e]=n}return t[e]}}();function c(t,e){for(var n=[],i={},r=0;r<t.length;r++){var o=t[r],a=e.base?o[0]+e.base:o[0],c={css:o[1],media:o[2],sourceMap:o[3]};i[a]?i[a].parts.push(c):n.push(i[a]={id:a,parts:[c]})}return n}function s(t,e){for(var n=0;n<t.length;n++){var i=t[n],o=r[i.id],a=0;if(o){for(o.refs++;a<o.parts.length;a++)o.parts[a](i.parts[a]);for(;a<i.parts.length;a++)o.parts.push(v(i.parts[a],e))}else{for(var c=[];a<i.parts.length;a++)c.push(v(i.parts[a],e));r[i.id]={id:i.id,refs:1,parts:c}}}}function l(t){var e=document.createElement("style");if(void 0===t.attributes.nonce){var i=n.nc;i&&(t.attributes.nonce=i)}if(Object.keys(t.attributes).forEach((function(n){e.setAttribute(n,t.attributes[n])})),"function"==typeof t.insert)t.insert(e);else{var r=a(t.insert||"head");if(!r)throw new Error("Couldn't find a style target. This probably means that the value for the 'insert' parameter is invalid.");r.appendChild(e)}return e}var u,f=(u=[],function(t,e){return u[t]=e,u.filter(Boolean).join("\n")});function h(t,e,n,i){var r=n?"":i.css;if(t.styleSheet)t.styleSheet.cssText=f(e,r);else{var o=document.createTextNode(r),a=t.childNodes;a[e]&&t.removeChild(a[e]),a.length?t.insertBefore(o,a[e]):t.appendChild(o)}}function p(t,e,n){var i=n.css,r=n.media,o=n.sourceMap;if(r&&t.setAttribute("media",r),o&&btoa&&(i+="\n/*# sourceMappingURL=data:application/json;base64,".concat(btoa(unescape(encodeURIComponent(JSON.stringify(o))))," */")),t.styleSheet)t.styleSheet.cssText=i;else{for(;t.firstChild;)t.removeChild(t.firstChild);t.appendChild(document.createTextNode(i))}}var d=null,g=0;function v(t,e){var n,i,r;if(e.singleton){var o=g++;n=d||(d=l(e)),i=h.bind(null,n,o,!1),r=h.bind(null,n,o,!0)}else n=l(e),i=p.bind(null,n,e),r=function(){!function(t){if(null===t.parentNode)return!1;t.parentNode.removeChild(t)}(n)};return i(t),function(e){if(e){if(e.css===t.css&&e.media===t.media&&e.sourceMap===t.sourceMap)return;i(t=e)}else r()}}t.exports=function(t,e){(e=e||{}).attributes="object"==typeof e.attributes?e.attributes:{},e.singleton||"boolean"==typeof e.singleton||(e.singleton=o());var n=c(t,e);return s(n,e),function(t){for(var i=[],o=0;o<n.length;o++){var a=n[o],l=r[a.id];l&&(l.refs--,i.push(l))}t&&s(c(t,e),e);for(var u=0;u<i.length;u++){var f=i[u];if(0===f.refs){for(var h=0;h<f.parts.length;h++)f.parts[h]();delete r[f.id]}}}}},function(t,e,n){"use strict";function i(t,e,n){if(t.getContext){void 0===n&&(n={unitSize:20,lineWidth:3,nodeRadius:4});var i=[],r=[],o=t.getContext("2d"),a=(window.devicePixelRatio||1)/(o.webkitBackingStorePixelRatio||o.mozBackingStorePixelRatio||o.msBackingStorePixelRatio||o.oBackingStorePixelRatio||o.backingStorePixelRatio||1),c=function(){var t,e,n="0123456789ABCDEF",i="";for(e=0;e<6;e++)t=Math.floor(Math.random()*n.length),i+=n.substring(t,t+1);return i},s=function(t){for(var e=i.length;e--&&i[e].id!==t;);return e},l=function(t,e){for(var n=e.length;n--&&e[n]!==t;);return n},u=function(t){if(!t)return-1;for(var e=t.length;e--&&(!t[e-1]||"/"!==t[e]||"|"!==t[e-1])&&(!t[e-2]||"_"!==t[e]||"|"!==t[e-2]););return e},f=function(t){if(!t)return-1;for(var e=t.length;e--&&(!t[e-1]||!t[e-2]||" "!==t[e]||"|"!==t[e-1]||"_"!==t[e-2]););return e},h=function(){var t;do{t=c()}while(-1!==s(t));return{id:t,color:"#".concat(t)}},p=function(t,e,n,i,r){o.strokeStyle=r,o.beginPath(),o.moveTo(t,e),o.lineTo(n,i),o.stroke()},d=function(t,e,i){p(t,e+n.unitSize/2,t+n.unitSize,e+n.unitSize/2,i)},g=function(t,e,i){p(t,e+n.unitSize/2,t,e-n.unitSize/2,i)},v=function(t,e,i){o.strokeStyle=i,g(t,e,i),o.beginPath(),o.arc(t,e,n.nodeRadius,0,2*Math.PI,!0),o.fill()},m=function(t,e,i){p(t+n.unitSize,e+n.unitSize/2,t,e-n.unitSize/2,i)},b=function(t,e,i){p(t,e+n.unitSize/2,t+n.unitSize,e-n.unitSize/2,i)};!function(){var i,c,s,l=0,u=e.length;for(i=0;i<u;i++)s=e[i].replace(/\s+/g," ").replace(/^\s+|\s+$/g,""),l=Math.max(s.replace(/(_|\s)/g,"").length,l),c=s.split(""),r.unshift(c);var f=l*n.unitSize,h=r.length*n.unitSize;t.width=f*a,t.height=h*a,t.style.width="".concat(f,"px"),t.style.height="".concat(h,"px"),o.lineWidth=n.lineWidth,o.lineJoin="round",o.lineCap="round",o.scale(a,a)}(),function(e){var r,o,c,s,p,S,w,x,y,z,B,C,k,M=-1,R=0,_=-1,T=0,j=!1;for(B=0,C=e[0].length;B<C;B++)"_"!==e[0][B]&&" "!==e[0][B]&&i.push(h());for(S=t.height/a-.5*n.unitSize,B=0,C=e.length;B<C;B++){p=.5*n.unitSize;var P=e[B],N=e[B+1],E=e[B-1];if(_=-1,k=P.filter((function(t){return" "!==t&&"_"!==t})).length,E){if(!j)for(o=0;o<R;o++)(E[o+1]&&"/"===E[o]&&"|"===E[o+1]||"_"===E[o]&&"|"===E[o+1]&&"/"===E[o+2])&&(y={id:i[_=o].id,color:i[_].color},i[_].id=i[_+1].id,i[_].color=i[_+1].color,i[_+1].id=y.id,i[_+1].color=y.color);T<k&&-1!==(x=l("*",P))&&-1===l("_",P)&&i.splice(x-1,0,h()),R>P.length&&-1!==(x=l("*",E))&&-1===l("_",P)&&-1===l("/",P)&&-1===l("\\",P)&&i.splice(x+1,1)}for(R=P.length,o=0,s=0,T=0,M=-1;o<P.length;){if(" "!==(r=P[o])&&"_"!==r&&++T,"/"===r&&P[o-1]&&"|"===P[o-1]&&-1!==(M=f(N))&&N.splice(M,1),-1!==M&&"/"===r&&o>M&&(P[o]="|",r="|")," "===r&&P[o+1]&&"_"===P[o+1]&&P[o-1]&&"|"===P[o-1]&&(P.splice(o,1),P[o]="/",r="/"),-1===_&&"/"===r&&P[o-1]&&"|"===P[o-1]&&i.splice(s,0,h()),("/"===r||"\\"===r)&&("/"!==r||-1!==u(N))&&-1!==(z=Math.max(l("|",P),l("*",P)))&&z<o-1){for(;" "===P[++z];);z===o&&(P[o]="|")}"*"===r&&E&&"\\"===E[s+1]&&i.splice(s+1,1)," "!==r&&++s,++o}for(k=P.filter((function(t){return" "!==t&&"_"!==t})).length,i.length>k&&i.splice(k,i.length-k),o=0;o<P.length;)if(r=P[o],c=P[o-1]," "!==P[o]){switch("_"!==r&&"/"!==r||"|"!==P[o-1]||"_"!==P[o-2]?j=!1:(j=!0,y=i.splice(o-2,1)[0],i.splice(o-1,0,y),P.splice(o-2,1),o-=1),w=i[o].color,r){case"_":d(p,S,w),p+=n.unitSize;break;case"*":v(p,S,w);break;case"|":g(p,S,w);break;case"/":!c||"/"!==c&&" "!==c||(p-=n.unitSize),b(p,S,w),p+=n.unitSize;break;case"\\":m(p,S,w)}++o}else P.splice(o,1),p+=n.unitSize;S-=n.unitSize}}(r)}}n.r(e),n.d(e,"default",(function(){return i}))},function(t,e,n){var i=n(10);"string"==typeof i&&(i=[[t.i,i,""]]);var r={insert:"head",singleton:!1};n(12)(i,r);i.locals&&(t.exports=i.locals)}]]);
//# sourceMappingURL=gitgraph.js.map