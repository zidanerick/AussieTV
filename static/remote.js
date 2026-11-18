/* AussieTV TV remote and keyboard navigation, v1.8. No external dependencies.
 * Native buttons/links/selects remain accessible for touch, mouse and keyboard.
 * Can be reused in a later container-hosted web UI. */
(function(){
 'use strict';
 const $=id=>document.getElementById(id);
 const focusable='button:not([disabled]),a[href],select:not([disabled]),input:not([disabled]),[tabindex]:not([tabindex="-1"])';
 const directions={ArrowUp:'up',ArrowDown:'down',ArrowLeft:'left',ArrowRight:'right'};
 const legacyKeys={37:'ArrowLeft',38:'ArrowUp',39:'ArrowRight',40:'ArrowDown',13:'Enter',27:'Escape',8:'Back',4:'Back',461:'Back',10009:'Back',427:'ChannelUp',428:'ChannelDown',33:'PageUp',34:'PageDown',179:'MediaPlayPause'};
 let modalReturn=null, textReturn=null, numeric='',numericTimer=null,hudTimer=null;
 let editingSelect=null,selectOriginalValue='';
 let lastSection='guide';
 function keyOf(event){
  const key=event.key;
  if(key && key!=='Unidentified' && key!=='Dead' && key!=='Process'){
   const alias={Select:'Enter',OK:'Enter',Accept:'Enter',NumpadEnter:'Enter',GoBack:'Back',BrowserBack:'Back',Return:'Back',XF86Back:'Back',Backspace:'Back',Esc:'Escape',Escape:'Escape',ChannelNext:'ChannelUp',ChannelPrevious:'ChannelDown',MediaPlay:'MediaPlayPause',MediaPause:'MediaPlayPause'};
   return alias[key]||key;
  }
  return legacyKeys[event.keyCode]||'';
 }
 function usable(el){
  if(!el || el.disabled || el.closest('[hidden],[aria-hidden="true"],[inert]'))return false;
  const box=el.getBoundingClientRect();
  if(!box.width||!box.height)return false;
  const style=window.getComputedStyle(el);
  return style.display!=='none'&&style.visibility!=='hidden';
 }
 function candidates(scope=document){return Array.from(scope.querySelectorAll(focusable)).filter(usable)}
 function scrollFocus(node){
  if(!node)return false;
  try{node.focus({preventScroll:true})}catch{node.focus()}
  if(document.activeElement!==node)return false;
  try{node.scrollIntoView({block:'nearest',inline:'nearest',behavior:'auto'})}catch{try{node.scrollIntoView(false)}catch{}}
  return true;
 }
 function firstFocusable(scope=document){return candidates(scope)[0]||null}
 function guideFocus(){
  if(!$('guide').querySelector('.channel-cell,.show-cell') && all.length){filter='All';buildFilters();buildGuide()}
  const active=Array.from($('guide').querySelectorAll('.channel-cell')).find(node=>node.dataset.ch===selected?.id);
  return active||$('guide').querySelector('.channel-cell,.show-cell')||$('watch');
 }
 function isDialogOpen(){return !$('remoteHelpPanel').hidden}
 function showHelp(open){
  const dialog=$('remoteHelpPanel');
  if(open){modalReturn=document.activeElement;dialog.hidden=false;scrollFocus($('remoteHelpClose'))}
  else{dialog.hidden=true;scrollFocus(usable(modalReturn)?modalReturn:($('remoteHelp')||guideFocus()));modalReturn=null}
 }
 function showText(open){
  if(open){textReturn=document.activeElement;toggleText(true);scrollFocus($('textPanel').querySelector('[data-textpage="100"]'))}
  else{toggleText(false);scrollFocus(usable(textReturn)?textReturn:$('textToggle'));textReturn=null}
 }
 function hideHud(){const hud=$('remoteHud');hud.hidden=true;hud.textContent='';}
 function notice(msg,ms=1800){
  const hud=$('remoteHud');hud.textContent=msg;hud.hidden=false;
  clearTimeout(hudTimer);hudTimer=setTimeout(hideHud,ms);
 }
 // In remote mode horizontal movement is deliberately kept inside a single
 // visual control group. A small diagonal distance must not jump to the video
 // controls or another guide row merely because its centre is closer.
 const horizontalZones='.top-actions,#filters,.player-bottom,.now-actions,#quickChannels,.guide-actions,.teletext-panel nav,.remote-help-box';
 function horizontalZone(el){return el.closest?.(horizontalZones)||null}
 function sameVisualRow(a,b){
  const overlap=Math.min(a.bottom,b.bottom)-Math.max(a.top,b.top);
  return overlap>=Math.max(2,Math.min(a.height,b.height)*.40);
 }
 function nearest(current,key,scope){
  const from=current.getBoundingClientRect();
  const fx=from.left+from.width/2,fy=from.top+from.height/2;
  const horizontal=(key==='ArrowLeft'||key==='ArrowRight');
  const zone=horizontal?horizontalZone(current):null;
  const list=candidates(scope).filter(el=>el!==current&&(!zone||zone.contains(el)));
  let best=null,score=Infinity;
  const direction=directions[key];
  for(const node of list){
   const rect=node.getBoundingClientRect();const x=rect.left+rect.width/2,y=rect.top+rect.height/2;
   const parallel=direction==='left'?fx-x:direction==='right'?x-fx:direction==='up'?fy-y:y-fy;
   if(parallel<3)continue;
   if(horizontal&&!sameVisualRow(from,rect))continue;
   const orthogonal=horizontal?Math.abs(y-fy):Math.abs(x-fx);
   const cross=horizontal?(from.height+rect.height)/2:(from.width+rect.width)/2;
   const penalty=orthogonal>cross?orthogonal-cross:0;
   const result=parallel+orthogonal*.45+penalty*2.8;
   if(result<score){score=result;best=node}
  }
  return best;
 }
 function beginSelect(select){
  if(!select||select.disabled)return;
  editingSelect=select;selectOriginalValue=select.value;
  select.classList.add('remote-select-editing');
  if(!select.hasAttribute('data-remote-base-label'))select.setAttribute('data-remote-base-label',select.getAttribute('aria-label')||'');
  select.setAttribute('aria-label',(select.getAttribute('aria-label')||select.title||'Selection')+'. Editing: up/down change, OK applies, Back cancels');
  notice('Editing '+(select.id==='region'?'region':'captions')+': ▲/▼ change, OK apply, Back cancel',3200);
 }
 function finishSelect(commit){
  if(!editingSelect)return;
  const node=editingSelect,changed=node.value!==selectOriginalValue;
  editingSelect=null;node.classList.remove('remote-select-editing');
  const base=node.getAttribute('data-remote-base-label');
  if(base!==null){if(base)node.setAttribute('aria-label',base);else node.removeAttribute('aria-label');node.removeAttribute('data-remote-base-label')}
  else if(node.id==='region'){node.removeAttribute('aria-label')}
  if(commit){
   if(changed)node.dispatchEvent(new Event('change',{bubbles:true}));
   notice('Selected '+node.options[node.selectedIndex]?.textContent,1200);
  }else{
   node.value=selectOriginalValue;
   notice('Selection cancelled',1100);
  }
 }
 function stepSelect(key){
  if(!editingSelect)return;
  const options=Array.from(editingSelect.options);
  let idx=editingSelect.selectedIndex,delta=key==='ArrowDown'?1:key==='ArrowUp'?-1:0;
  if(!delta)return;
  do {idx+=delta} while(idx>=0&&idx<options.length&&options[idx].disabled);
  if(idx>=0&&idx<options.length)editingSelect.selectedIndex=idx;
  notice((editingSelect.options[editingSelect.selectedIndex]?.textContent||'')+' · OK to apply · Back to cancel',2500);
 }
 // A D-pad must never hit a dead end. Spatial search runs inside the current
 // logical section first; at a boundary, an explicit section graph takes over.
 // This also avoids wide video/EPG geometry pulling focus diagonally.
 const sections={
  filters:'#filters', top:'.top-actions', player:'.player-bottom',
  now:'.now-actions', quick:'#quickChannels', guideActions:'.guide-actions',
  guide:'#guide'
 };
 const transitions={
  filters:{left:'top',right:'guide',up:'top',down:'guide'},
  top:{left:'filters',right:'player',up:'filters',down:'player'},
  player:{left:'filters',right:'now',up:'top',down:'quick'},
  now:{left:'player',right:'quick',up:'top',down:'quick'},
  quick:{left:'filters',right:'guideActions',up:'now',down:'guideActions'},
  guideActions:{left:'quick',right:'guide',up:'quick',down:'guide'},
  guide:{left:'filters',right:'guideActions',up:'guideActions',down:'guideActions'}
 };
 function sectionOf(element){
  if(!element?.closest)return null;
  for(const [name,selector] of Object.entries(sections))if(element.closest(selector))return name;
  // Treat the retry button (outside player-bottom) as part of the player.
  if(element.closest('#player'))return 'player';
  return null;
 }
 function sectionElement(name){return document.querySelector(sections[name]||'#guide')}
 function nearestAny(current,scope){
  const nodes=candidates(scope);if(!nodes.length)return null;
  if(!current?.getBoundingClientRect)return nodes[0];
  const a=current.getBoundingClientRect(),x=a.left+a.width/2,y=a.top+a.height/2;
  nodes.sort((m,n)=>{
   const d=r=>{const t=r.getBoundingClientRect();return Math.abs(t.left+t.width/2-x)+Math.abs(t.top+t.height/2-y)};
   return d(m)-d(n);
  });
  return nodes[0];
 }
 function exitSection(active,key,section){
  const dir=directions[key];if(!dir)return false;
  const target=transitions[section]?.[dir];
  if(!target)return false;
  // A target section can be empty while the EPG loads or in a narrow layout.
  const order=[target,'guide','filters','top','quick','now','player','guideActions'];
  for(const name of [...new Set(order)]){
   const scope=sectionElement(name);
   if(!scope)continue;
   const node=nearestAny(active,scope);
   if(node&&node!==active)return scrollFocus(node);
  }
  return false;
 }
 function move(key){
  const overlay=isDialogOpen()||!$('textPanel').hidden;
  const scope=isDialogOpen()?$('remoteHelpPanel'):(!$('textPanel').hidden?$('textPanel'):document);
  let active=document.activeElement;
  if(!usable(active)||!scope.contains(active)||!active.matches(focusable)){
   // A dynamic redraw can delete the focused element. Recover in its former
   // section rather than sending the viewer back to the beginning of the guide.
   const preferred=overlay?firstFocusable(scope):firstFocusable(sectionElement(lastSection));
   const playerControl=active===$('video')?firstFocusable(document.querySelector('.player-bottom')):null;
   scrollFocus(playerControl||preferred||guideFocus());return;
  }
  if(overlay){
   const next=nearest(active,key,scope);
   if(next){scrollFocus(next);return}
   const list=candidates(scope);
   if(list.length>1)scrollFocus(key==='ArrowLeft'||key==='ArrowUp'?list[list.length-1]:list[0]);
   return;
  }
  const section=sectionOf(active);
  if(section==='guide'){
   // Preserve logical programme-grid movement, but permit leaving every edge.
   moveGuideFocus(active,key);
   if(document.activeElement!==active){scrollFocus(document.activeElement);return}
   exitSection(active,key,'guide');return;
  }
  const container=section&&sectionElement(section);
  const next=container?nearest(active,key,container):nearest(active,key,scope);
  if(next){scrollFocus(next);return}
  if(section&&exitSection(active,key,section))return;
  scrollFocus(guideFocus());
 }
 function back(){
  if(editingSelect){finishSelect(false);return}
  if(isDialogOpen()){showHelp(false);return}
  if(!$('textPanel').hidden){showText(false);return}
  if(document.fullscreenElement||document.webkitFullscreenElement){try{(document.exitFullscreen||document.webkitExitFullscreen).call(document)}catch{}return}
  const active=document.activeElement;
  if(active?.tagName==='SELECT'){scrollFocus(guideFocus());return}
  if(active?.closest('#guide')){scrollFocus($('watch'));return}
  scrollFocus(guideFocus());
 }
 function numberKey(key){
  if(!/^[0-9]$/.test(key))return false;
  clearTimeout(numericTimer);numeric=(numeric+key).slice(-3);notice('Channel '+numeric,1700);
  numericTimer=setTimeout(()=>{
   const match=all.find(ch=>String(ch.number).replace(/\D/g,'')===numeric);
   if(match){
    if((filter==='Favourites'&&!isFavourite(match))||(filter!=='All'&&filter!=='Favourites'&&filter!==match.network)){filter='All';buildFilters()}
    selectChannel(match,true);notice(match.name);
    scrollFocus(guideFocus());
   }else{notice('Channel '+numeric+' not found')}
   numeric='';
  },1050);
  return true;
 }
 function toggleMedia(){
  const video=$('video');
  if(!selected)return;
  if(video.paused){if(!video.currentSrc)playChannel();else video.play().catch(()=>notice('Select Watch live to begin playback'))}
  else video.pause();
  syncMediaControls();
 }
 function syncMediaControls(){
  const v=$('video');$('playPause').textContent=v.paused?'▶ Play':'⏸ Pause';$('mute').textContent=v.muted?'◖ Muted':'◖ Sound';$('mute').setAttribute('aria-pressed',String(v.muted));
 }
 function fullscreenToggle(){
  const node=$('player');
  if(document.fullscreenElement||document.webkitFullscreenElement){try{(document.exitFullscreen||document.webkitExitFullscreen).call(document)}catch{}return}
  const fn=node.requestFullscreen||node.webkitRequestFullscreen||node.msRequestFullscreen;
  if(fn){try{const result=fn.call(node);if(result?.catch)result.catch(()=>notice('Fullscreen unavailable in this browser'))}catch{notice('Fullscreen unavailable in this browser')}}
  else if($('video').webkitEnterFullscreen){try{$('video').webkitEnterFullscreen()}catch{notice('Fullscreen unavailable')}}
  else notice('Fullscreen unavailable in this browser');
 }
 function activateFocused(){
  const active=document.activeElement;
  if(usable(active)&&active.matches('button,a')){active.click();return}
  if(usable(active)&&active.matches('select')){beginSelect(active);return}
  if(usable(active)&&active.matches('[role="button"]')){active.click();return}
  scrollFocus(guideFocus());
 }
 function handle(event){
  if(event.ctrlKey||event.metaKey||event.altKey)return;
  const key=keyOf(event);
  if(!key)return;
  // Never convert genuine editing keystrokes into remote commands.
  const source=event.target;
  if(source?.matches?.('input,textarea,[contenteditable="true"]'))return;
  if(/^(ArrowUp|ArrowDown|ArrowLeft|ArrowRight|Enter|Back|Escape|PageUp|PageDown|ChannelUp|ChannelDown|MediaPlayPause|Home|h|H|g|G|m|M|c|C|t|T|f|F|[0-9])$/.test(key))
   document.body.classList.add('remote-navigation');
  const overlay=isDialogOpen()||!$('textPanel').hidden;
  if(key==='Back'||key==='Escape'){
   event.preventDefault();event.stopImmediatePropagation();back();return;
  }
  if(directions[key]){
   event.preventDefault();event.stopImmediatePropagation();
   if(editingSelect){stepSelect(key);return}
   move(key);return;
  }
  if(key==='Enter'||key===' '){
   event.preventDefault();event.stopImmediatePropagation();
   if(editingSelect){finishSelect(true);return}
   if(key===' '&&!source?.matches?.('button,a,[role="button"],select')){toggleMedia();return}
   activateFocused();return;
  }
  if(editingSelect){finishSelect(false)}
  if(overlay){
   if((key==='t'||key==='T')&&!isDialogOpen()){event.preventDefault();showText(false)}
   if(key==='h'||key==='H'){event.preventDefault();showHelp(!isDialogOpen())}
   return;
  }
  if(key==='PageUp'||key==='ChannelUp'||key==='PageDown'||key==='ChannelDown'){
   event.preventDefault();navigate((key==='PageUp'||key==='ChannelUp')?-1:1);notice(selected?.name||'Channel');return;
  }
  if(key==='MediaPlayPause'){event.preventDefault();toggleMedia();return}
  if(key==='Home'||key==='g'||key==='G'){event.preventDefault();scrollFocus(guideFocus());return}
  if(key==='m'||key==='M'){event.preventDefault();scrollFocus(firstFocusable($('filters')));return}
  if(key==='h'||key==='H'){event.preventDefault();showHelp(true);return}
  if(key==='t'||key==='T'){event.preventDefault();showText(true);return}
  if(key==='c'||key==='C'){event.preventDefault();toggleCaptions();return}
  if(key==='f'||key==='F'){event.preventDefault();fullscreenToggle();return}
  if(numberKey(key)){event.preventDefault();return}
 }
 // Capture before built-in browser arrow scrolling and native media handlers;
 // normal click/touch events still activate every existing control.
 document.addEventListener('keydown',handle,true);
 document.addEventListener('focusin',event=>{
  if(editingSelect&&event.target!==editingSelect)finishSelect(false);
  const section=sectionOf(event.target);if(section)lastSection=section;
 });
 $('remoteHelp').addEventListener('click',()=>showHelp(true));
 $('remoteHelpClose').addEventListener('click',()=>showHelp(false));
 $('textToggle').addEventListener('click',()=>{
  // Existing click handler runs before this one; sync focus with the state.
  if(!$('textPanel').hidden){textReturn=document.activeElement;scrollFocus($('textPanel').querySelector('[data-textpage="100"]'))}
  else scrollFocus(usable(textReturn)?textReturn:$('textToggle'));
 });
 $('textClose').addEventListener('click',()=>scrollFocus(usable(textReturn)?textReturn:$('textToggle')));
 $('fullscreen').onclick=fullscreenToggle;
 $('playPause').addEventListener('click',toggleMedia);
 $('mute').addEventListener('click',()=>{$('video').muted=!$('video').muted;syncMediaControls()});
 for(const event of ['play','pause','volumechange','loadedmetadata'])$('video').addEventListener(event,syncMediaControls);
 syncMediaControls();
 // Expose pure-ish primitives to the browser integration tests without
 // creating a separate dependency on a browser framework.
 window.AussieTVRemote={keyOf,nearest,candidates,move,back,showHelp,showText,guideFocus,sameVisualRow,horizontalZone,sectionOf,exitSection,transitions};
})();
