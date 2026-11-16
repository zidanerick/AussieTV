/* AussieTV TV remote and keyboard navigation, v1.7. No external dependencies.
 * Native buttons/links/selects remain accessible for touch, mouse and keyboard.
 * Can be reused in a later container-hosted web UI. */
(function(){
 'use strict';
 const $=id=>document.getElementById(id);
 const focusable='button:not([disabled]),a[href],select:not([disabled]),input:not([disabled]),[tabindex]:not([tabindex="-1"])';
 const directions={ArrowUp:'up',ArrowDown:'down',ArrowLeft:'left',ArrowRight:'right'};
 const legacyKeys={37:'ArrowLeft',38:'ArrowUp',39:'ArrowRight',40:'ArrowDown',13:'Enter',27:'Escape',8:'Back',4:'Back',461:'Back',10009:'Back',427:'ChannelUp',428:'ChannelDown',33:'PageUp',34:'PageDown',179:'MediaPlayPause'};
 let modalReturn=null, textReturn=null, numeric='',numericTimer=null,hudTimer=null;
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
 function nearest(current,key,scope){
  const list=candidates(scope).filter(el=>el!==current);
  const from=current.getBoundingClientRect();
  const fx=from.left+from.width/2,fy=from.top+from.height/2;
  let best=null,score=Infinity;
  const direction=directions[key];
  for(const node of list){
   const rect=node.getBoundingClientRect();const x=rect.left+rect.width/2,y=rect.top+rect.height/2;
   const parallel=direction==='left'?fx-x:direction==='right'?x-fx:direction==='up'?fy-y:y-fy;
   if(parallel<3)continue;
   const orthogonal=direction==='left'||direction==='right'?Math.abs(y-fy):Math.abs(x-fx);
   const cross=direction==='left'||direction==='right'?(from.height+rect.height)/2:(from.width+rect.width)/2;
   // Strongly prefer items in the same row/column, while allowing the
   // remote to move between the sidebar, controls, player, and EPG.
   const penalty=orthogonal>cross?orthogonal-cross:0;
   const result=parallel+orthogonal*.45+penalty*2.8;
   if(result<score){score=result;best=node}
  }
  return best;
 }
 function move(key){
  const scope=isDialogOpen()?$('remoteHelpPanel'):(!$('textPanel').hidden?$('textPanel'):document);
  let active=document.activeElement;
  if(!usable(active)||!scope.contains(active)||!active.matches(focusable)){scrollFocus(isDialogOpen()||!$('textPanel').hidden?firstFocusable(scope):guideFocus());return}
  if(!isDialogOpen()&&$('textPanel').hidden&&active.matches('#guide .channel-cell,#guide .show-cell')){
   const initial=active;
   moveGuideFocus(active,key);
   if(document.activeElement!==initial){
    const destination=document.activeElement;
    try{destination.scrollIntoView({block:'nearest',inline:'nearest'})}catch{}
    return;
   }
  }
  const next=nearest(active,key,scope);
  if(next)scrollFocus(next);
 }
 function back(){
  if(isDialogOpen()){showHelp(false);return}
  if(!$('textPanel').hidden){showText(false);return}
  if(document.fullscreenElement||document.webkitFullscreenElement){try{(document.exitFullscreen||document.webkitExitFullscreen).call(document)}catch{}return}
  const active=document.activeElement;
  if(active?.tagName==='SELECT'){scrollFocus($('remoteHelp'));return}
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
  if(usable(active)&&active.matches('select')){active.focus();return}
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
   // Browser-native dropdowns must retain their own arrow key handling.
   if(source?.tagName==='SELECT'&&!overlay)return;
   event.preventDefault();event.stopImmediatePropagation();move(key);return;
  }
  if(key==='Enter'||key===' '){
   if(source?.tagName==='SELECT')return;
   if(key===' '&&!source?.matches?.('button,a,[role="button"]')){event.preventDefault();toggleMedia();return}
   event.preventDefault();event.stopImmediatePropagation();activateFocused();return;
  }
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
 window.AussieTVRemote={keyOf,nearest,candidates,move,back,showHelp,showText,guideFocus};
})();
