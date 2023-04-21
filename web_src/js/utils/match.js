import emojis from '../../../assets/emoji.json';

const maxMatches = 6;

export function matchEmoji(q) {
  const query = q.toLowerCase().replaceAll('_', ' ');

  const results = new Map();
  for (const {aliases} of emojis) {
    const mainAlias = aliases[0];
    for (const alias of aliases) {
      const index = alias.replaceAll('_', ' ').indexOf(query);
      if (index === -1) continue;
      const existing = results.get(mainAlias);
      results.set(mainAlias, existing ? existing - index : index);
    }
  }

  const sortedMap = new Map([...results.entries()].sort((a, b) => a[1] - b[1]));
  return Array.from(sortedMap.keys()).slice(0, maxMatches);
}

export function matchMention(q) {
  const query = q.toLowerCase();

  const results = new Map();
  for (const obj of window.config.tributeValues) {
    const index = obj.key.toLowerCase().indexOf(query);
    if (index === -1) continue;
    const existing = results.get(obj);
    results.set(obj, existing ? existing - index : index);
  }

  const sortedMap = new Map([...results.entries()].sort((a, b) => a[1] - b[1]));
  return Array.from(sortedMap.keys()).slice(0, maxMatches);
}
