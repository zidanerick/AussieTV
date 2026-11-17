'use strict';
const {test}=require('node:test');
const assert=require('node:assert/strict');
const fs=require('node:fs');
const path=require('node:path');
const vm=require('node:vm');
const code=fs.readFileSync(path.join(__dirname,'..','static','remote.js'),'utf8');
const fake={addEventListener(){},setAttribute(){},textContent:'',paused:true,muted:false};
const document={getElementById(){return fake},addEventListener(){},body:{classList:{add(){}}}};
const window={getComputedStyle(){return {display:'block',visibility:'visible'}}};
vm.runInNewContext(code,{document,window,setTimeout,clearTimeout});
const r=window.AussieTVRemote;
test('TV remote and keyboard key aliases',()=>{
 for(const [event,want] of [
  [{key:'OK'},'Enter'],[{key:'Select'},'Enter'],[{key:'NumpadEnter'},'Enter'],
  [{key:'Back'},'Back'],[{key:'BrowserBack'},'Back'],[{key:'XF86Back'},'Back'],
  [{key:'Unidentified',keyCode:10009},'Back'],[{key:'Unidentified',keyCode:461},'Back'],[{key:'Backspace'},'Back'],
  [{key:'Unidentified',keyCode:4},'Back'],[{key:'Unidentified',keyCode:37},'ArrowLeft'],
  [{key:'Unidentified',keyCode:39},'ArrowRight'],[{key:'Unidentified',keyCode:427},'ChannelUp'],
  [{key:'Unidentified',keyCode:428},'ChannelDown'],[{key:'PageDown'},'PageDown'],
  [{key:'8'},'8'],[{key:'Unidentified',keyCode:33},'PageUp']
 ])assert.equal(r.keyOf(event),want,JSON.stringify(event));
});
function rect(left,top,width=80,height=40){return {left,top,width,height,right:left+width,bottom:top+height}}
function element(box){return {
 disabled:false,closest(){return false},getBoundingClientRect(){return box}
}}
test('spatial navigation selects nearest valid candidate',()=>{
 const from=element(rect(100,100));
 const above=element(rect(102,10));
 const below=element(rect(100,230));
 const right=element(rect(230,100));
 const left=element(rect(-30,100));
 const fartherRight=element(rect(300,100));
 const scope={querySelectorAll(){return [from,above,below,right,left,fartherRight]}};
 assert.equal(r.nearest(from,'ArrowUp',scope),above);
 assert.equal(r.nearest(from,'ArrowDown',scope),below);
 assert.equal(r.nearest(from,'ArrowRight',scope),right);
 assert.equal(r.nearest(from,'ArrowLeft',scope),left);
});
test('disabled, hidden, and zero-size elements are never navigation targets',()=>{
 const start=element(rect(0,0));const hidden=element(rect(100,0));hidden.closest=()=>true;
 const disabled=element(rect(150,0));disabled.disabled=true;
 const zero=element(rect(250,0,0,0));const real=element(rect(400,0));
 const scope={querySelectorAll(){return [start,hidden,disabled,zero,real]}};
 assert.equal(r.nearest(start,'ArrowRight',scope),real);
});

test('left and right never jump diagonally to another visual row',()=>{
 const from=element(rect(100,100));
 const nextRow=element(rect(180,180));
 const aligned=element(rect(360,104));
 const scope={querySelectorAll(){return [from,nextRow,aligned]}};
 assert.equal(r.nearest(from,'ArrowRight',scope),aligned);
 assert.equal(r.nearest(aligned,'ArrowLeft',scope),from);
 const onlyDiagonal={querySelectorAll(){return [from,nextRow]}};
 assert.equal(r.nearest(from,'ArrowRight',onlyDiagonal),null);
});
