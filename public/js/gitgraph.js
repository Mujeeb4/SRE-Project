(window.webpackJsonp=window.webpackJsonp||[]).push([[0],[,,,,,,,,,function(t,n,e){(t.exports=e(10)(!1)).push([t.i,"#git-graph-container, #rel-container {float:left;}\r\n#rel-container {max-width:30%; overflow-x:auto;}\r\n#git-graph-container {overflow-x:auto; width:100%}\r\n#git-graph-container li {list-style-type:none;height:20px;line-height:20px; white-space:nowrap;}\r\n#git-graph-container li .node-relation {font-family:'Bitstream Vera Sans Mono', 'Courier', monospace;}\r\n#git-graph-container li .author {color:#666666;}\r\n#git-graph-container li .time {color:#999999;font-size:80%}\r\n#git-graph-container li a {color:#000000;}\r\n#git-graph-container li a:hover {text-decoration:underline;}\r\n#git-graph-container li a em {color:#BB0000;border-bottom:1px dotted #BBBBBB;text-decoration:none;font-style:normal;}\r\n#rev-container {width:100%}\r\n#rev-list {margin:0;padding:0 5px 0 5px;min-width:95%}\r\n#graph-raw-list {margin:0px;}\r\n",""])},function(t,n,e){"use strict";t.exports=function(t){var n=[];return n.toString=function(){return this.map((function(n){var e=function(t,n){var e=t[1]||"",i=t[3];if(!i)return e;if(n&&"function"==typeof btoa){var r=(a=i,c=btoa(unescape(encodeURIComponent(JSON.stringify(a)))),l="sourceMappingURL=data:application/json;charset=utf-8;base64,".concat(c),"/*# ".concat(l," */")),o=i.sources.map((function(t){return"/*# sourceURL=".concat(i.sourceRoot).concat(t," */")}));return[e].concat(o).concat([r]).join("\n")}var a,c,l;return[e].join("\n")}(n,t);return n[2]?"@media ".concat(n[2],"{").concat(e,"}"):e})).join("")},n.i=function(t,e){"string"==typeof t&&(t=[[null,t,""]]);for(var i={},r=0;r<this.length;r++){var o=this[r][0];null!=o&&(i[o]=!0)}for(var a=0;a<t.length;a++){var c=t[a];null!=c[0]&&i[c[0]]||(e&&!c[2]?c[2]=e:e&&(c[2]="(".concat(c[2],") and (").concat(e,")")),n.push(c))}},n}},function(t,n,e){"use strict";var i,r={},o=function(){return void 0===i&&(i=Boolean(window&&document&&document.all&&!window.atob)),i},a=function(){var t={};return function(n){if(void 0===t[n]){var e=document.querySelector(n);if(window.HTMLIFrameElement&&e instanceof window.HTMLIFrameElement)try{e=e.contentDocument.head}catch(t){e=null}t[n]=e}return t[n]}}();function c(t,n){for(var e=[],i={},r=0;r<t.length;r++){var o=t[r],a=n.base?o[0]+n.base:o[0],c={css:o[1],media:o[2],sourceMap:o[3]};i[a]?i[a].parts.push(c):e.push(i[a]={id:a,parts:[c]})}return e}function l(t,n){for(var e=0;e<t.length;e++){var i=t[e],o=r[i.id],a=0;if(o){for(o.refs++;a<o.parts.length;a++)o.parts[a](i.parts[a]);for(;a<i.parts.length;a++)o.parts.push(v(i.parts[a],n))}else{for(var c=[];a<i.parts.length;a++)c.push(v(i.parts[a],n));r[i.id]={id:i.id,refs:1,parts:c}}}}function s(t){var n=document.createElement("style");if(void 0===t.attributes.nonce){var i=e.nc;i&&(t.attributes.nonce=i)}if(Object.keys(t.attributes).forEach((function(e){n.setAttribute(e,t.attributes[e])})),"function"==typeof t.insert)t.insert(n);else{var r=a(t.insert||"head");if(!r)throw new Error("Couldn't find a style target. This probably means that the value for the 'insert' parameter is invalid.");r.appendChild(n)}return n}var u,f=(u=[],function(t,n){return u[t]=n,u.filter(Boolean).join("\n")});function h(t,n,e,i){var r=e?"":i.css;if(t.styleSheet)t.styleSheet.cssText=f(n,r);else{var o=document.createTextNode(r),a=t.childNodes;a[n]&&t.removeChild(a[n]),a.length?t.insertBefore(o,a[n]):t.appendChild(o)}}function p(t,n,e){var i=e.css,r=e.media,o=e.sourceMap;if(r&&t.setAttribute("media",r),o&&btoa&&(i+="\n/*# sourceMappingURL=data:application/json;base64,".concat(btoa(unescape(encodeURIComponent(JSON.stringify(o))))," */")),t.styleSheet)t.styleSheet.cssText=i;else{for(;t.firstChild;)t.removeChild(t.firstChild);t.appendChild(document.createTextNode(i))}}var d=null,g=0;function v(t,n){var e,i,r;if(n.singleton){var o=g++;e=d||(d=s(n)),i=h.bind(null,e,o,!1),r=h.bind(null,e,o,!0)}else e=s(n),i=p.bind(null,e,n),r=function(){!function(t){if(null===t.parentNode)return!1;t.parentNode.removeChild(t)}(e)};return i(t),function(n){if(n){if(n.css===t.css&&n.media===t.media&&n.sourceMap===t.sourceMap)return;i(t=n)}else r()}}t.exports=function(t,n){(n=n||{}).attributes="object"==typeof n.attributes?n.attributes:{},n.singleton||"boolean"==typeof n.singleton||(n.singleton=o());var e=c(t,n);return l(e,n),function(t){for(var i=[],o=0;o<e.length;o++){var a=e[o],s=r[a.id];s&&(s.refs--,i.push(s))}t&&l(c(t,n),n);for(var u=0;u<i.length;u++){var f=i[u];if(0===f.refs){for(var h=0;h<f.parts.length;h++)f.parts[h]();delete r[f.id]}}}}},function(t,n,e){"use strict";function i(t,n,e){if(t.getContext){void 0===e&&(e={unitSize:20,lineWidth:3,nodeRadius:4});var i=[],r=[],o=t.getContext("2d"),a=(window.devicePixelRatio||1)/(o.webkitBackingStorePixelRatio||o.mozBackingStorePixelRatio||o.msBackingStorePixelRatio||o.oBackingStorePixelRatio||o.backingStorePixelRatio||1),c=function(){var t,n,e="0123456789ABCDEF",i="";for(n=0;n<6;n++)t=Math.floor(Math.random()*e.length),i+=e.substring(t,t+1);return i},l=function(t){for(var n=i.length;n--&&i[n].id!==t;);return n},s=function(t,n){for(var e=n.length;e--&&n[e]!==t;);return e},u=function(t){if(!t)return-1;for(var n=t.length;n--&&(!t[n-1]||"/"!==t[n]||"|"!==t[n-1])&&(!t[n-2]||"_"!==t[n]||"|"!==t[n-2]););return n},f=function(t){if(!t)return-1;for(var n=t.length;n--&&(!t[n-1]||!t[n-2]||" "!==t[n]||"|"!==t[n-1]||"_"!==t[n-2]););return n},h=function(){var t;do{t=c()}while(-1!==l(t));return{id:t,color:"#".concat(t)}},p=function(t,n,e,i,r){o.strokeStyle=r,o.beginPath(),o.moveTo(t,n),o.lineTo(e,i),o.stroke()},d=function(t,n,i){p(t,n+e.unitSize/2,t+e.unitSize,n+e.unitSize/2,i)},g=function(t,n,i){p(t,n+e.unitSize/2,t,n-e.unitSize/2,i)},v=function(t,n,i){o.strokeStyle=i,g(t,n,i),o.beginPath(),o.arc(t,n,e.nodeRadius,0,2*Math.PI,!0),o.fill()},m=function(t,n,i){p(t+e.unitSize,n+e.unitSize/2,t,n-e.unitSize/2,i)},b=function(t,n,i){p(t,n+e.unitSize/2,t+e.unitSize,n-e.unitSize/2,i)};!function(){var i,c,l,s=0,u=n.length;for(i=0;i<u;i++)l=n[i].replace(/\s+/g," ").replace(/^\s+|\s+$/g,""),s=Math.max(l.replace(/(_|\s)/g,"").length,s),c=l.split(""),r.unshift(c);var f=s*e.unitSize,h=r.length*e.unitSize;t.width=f*a,t.height=h*a,t.style.width="".concat(f,"px"),t.style.height="".concat(h,"px"),o.lineWidth=e.lineWidth,o.lineJoin="round",o.lineCap="round",o.scale(a,a)}(),function(n){var r,o,c,l,p,S,w,x,y,z,B,k,C,M=-1,R=0,_=-1,P=0,T=!1;for(B=0,k=n[0].length;B<k;B++)"_"!==n[0][B]&&" "!==n[0][B]&&i.push(h());for(S=t.height/a-.5*e.unitSize,B=0,k=n.length;B<k;B++){p=.5*e.unitSize;var j=n[B],N=n[B+1],E=n[B-1];if(_=-1,C=j.filter((function(t){return" "!==t&&"_"!==t})).length,E){if(!T)for(o=0;o<R;o++)(E[o+1]&&"/"===E[o]&&"|"===E[o+1]||"_"===E[o]&&"|"===E[o+1]&&"/"===E[o+2])&&(y={id:i[_=o].id,color:i[_].color},i[_].id=i[_+1].id,i[_].color=i[_+1].color,i[_+1].id=y.id,i[_+1].color=y.color);P<C&&-1!==(x=s("*",j))&&-1===s("_",j)&&i.splice(x-1,0,h()),R>j.length&&-1!==(x=s("*",E))&&-1===s("_",j)&&-1===s("/",j)&&-1===s("\\",j)&&i.splice(x+1,1)}for(R=j.length,o=0,l=0,P=0,M=-1;o<j.length;){if(" "!==(r=j[o])&&"_"!==r&&++P,"/"===r&&j[o-1]&&"|"===j[o-1]&&-1!==(M=f(N))&&N.splice(M,1),-1!==M&&"/"===r&&o>M&&(j[o]="|",r="|")," "===r&&j[o+1]&&"_"===j[o+1]&&j[o-1]&&"|"===j[o-1]&&(j.splice(o,1),j[o]="/",r="/"),-1===_&&"/"===r&&j[o-1]&&"|"===j[o-1]&&i.splice(l,0,h()),("/"===r||"\\"===r)&&("/"!==r||-1!==u(N))&&-1!==(z=Math.max(s("|",j),s("*",j)))&&z<o-1){for(;" "===j[++z];);z===o&&(j[o]="|")}"*"===r&&E&&"\\"===E[l+1]&&i.splice(l+1,1)," "!==r&&++l,++o}for(C=j.filter((function(t){return" "!==t&&"_"!==t})).length,i.length>C&&i.splice(C,i.length-C),o=0;o<j.length;)if(r=j[o],c=j[o-1]," "!==j[o]){switch("_"!==r&&"/"!==r||"|"!==j[o-1]||"_"!==j[o-2]?T=!1:(T=!0,y=i.splice(o-2,1)[0],i.splice(o-1,0,y),j.splice(o-2,1),o-=1),w=i[o].color,r){case"_":d(p,S,w),p+=e.unitSize;break;case"*":v(p,S,w);break;case"|":g(p,S,w);break;case"/":!c||"/"!==c&&" "!==c||(p-=e.unitSize),b(p,S,w),p+=e.unitSize;break;case"\\":m(p,S,w)}++o}else j.splice(o,1),p+=e.unitSize;S-=e.unitSize}}(r)}}e.r(n),e.d(n,"default",(function(){return i}))},function(t,n,e){var i=e(9);"string"==typeof i&&(i=[[t.i,i,""]]);var r={insert:"head",singleton:!1};e(11)(i,r);i.locals&&(t.exports=i.locals)}]]);
//# sourceMappingURL=gitgraph.js.map