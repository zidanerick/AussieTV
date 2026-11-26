import {DiscordSDK} from '@discord/embedded-app-sdk';
import {nowNext, officialLink} from './guide.mjs';
const $ = id => document.getElementById(id);
const embedded = new URLSearchParams(location.search).has('frame_id');
const prefix = embedded ? '/.proxy' : '';
let sdk = null, generation = 0;
const status = $('status');
async function getJSON(path) {
  const response = await fetch(prefix + path, {signal:AbortSignal.timeout(30000)});
  if (!response.ok) throw new Error('Guide service HTTP '+response.status);
  return response.json();
}
async function connect() {
  if (!embedded) return;
  const {applicationId} = await getJSON('/api/config');
  const candidate = new DiscordSDK(applicationId);
  let timer;
  try {
    await Promise.race([candidate.ready(),new Promise((_,reject)=>{timer=setTimeout(()=>reject(new Error('Discord SDK connection timed out')),10000)})]);
    sdk = candidate;
  } finally { clearTimeout(timer); }
}
function text(tag, value, cls) {
  const el=document.createElement(tag);el.textContent=value;
  if(cls)el.className=cls;
  return el;
}
async function reload() {
  const ticket=++generation;
  status.textContent='Loading guide…';
  $('guide').replaceChildren();
  try {
    const guide=await getJSON('/api/guide?region='+encodeURIComponent($('region').value));
    if(ticket!==generation)return;
    for(const channel of guide.channels || []) {
      const card=text('article','','channel');
      const {current,next}=nowNext(channel.programs);
      card.append(text('h2',channel.number+' · '+channel.name),text('p','Now: '+current),text('p','Next: '+next,'next'));
      const link=officialLink(channel);
      if(link){
        const button=text('button','Open official player');
        button.disabled=embedded&&!sdk;
        button.onclick=async()=>{
          try {
            if(sdk) await sdk.commands.openExternalLink({url:link});
            else if(!embedded) window.open(link,'_blank','noopener,noreferrer');
          }catch{status.textContent='Could not open the broadcaster link.'}
        };
        card.append(button);
      }
      $('guide').append(card);
    }
    status.textContent=(guide.warning||guide.channels.length+' channels · '+guide.region)+(embedded&&!sdk?' · Discord connection unavailable':'');
  } catch {
    if(ticket===generation)status.textContent='Unable to load listings. Try Refresh.';
  }
}
$('refresh').onclick=reload;
$('region').onchange=reload;
try{await connect()}catch{status.textContent='Discord connection unavailable.'}
await reload();
setInterval(()=>{if(!document.hidden)reload()},60000);
