export function nowNext(programs, now = Date.now()/1000) {
  const list = Array.isArray(programs) ? programs.filter(p=>p && Number.isFinite(p.start) && Number.isFinite(p.end) && p.end>p.start && typeof p.title==='string') : [];
  return {
    current:list.find(p=>p.start<=now && now<p.end)?.title || 'No current listing',
    next:list.filter(p=>p.start>now).sort((a,b)=>a.start-b.start)[0]?.title || 'No upcoming listing'
  };
}
const official = Object.freeze({
  ABC:'https://iview.abc.net.au/collection/959',
  SBS:'https://www.sbs.com.au/ondemand/live',
  Seven:'https://7plus.com.au/live-tv',
  Nine:'https://www.9now.com.au/live/channel-9',
  Ten:'https://10.com.au/live'
});
export function officialLink(channel) {
  return Object.hasOwn(official,channel.network) && channel.official===official[channel.network] ? channel.official : null;
}
