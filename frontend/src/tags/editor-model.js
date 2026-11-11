import { canonicalTag } from '../resource-tags.js';

export function normalizeTagSelection(values) {
  const seen = new Set();
  return (values ?? []).reduce((tags, value) => {
    const name = String(value ?? '').trim();
    const key = canonicalTag(name);
    if (!key || seen.has(key)) return tags;
    seen.add(key);
    tags.push(name);
    return tags;
  }, []);
}

export function tagEditorSuggestions(catalog, selectedTags, query = '') {
  const selected = new Set(normalizeTagSelection(selectedTags).map(canonicalTag));
  const normalizedQuery = canonicalTag(query);
  return (catalog ?? []).filter((tag) => (
    !selected.has(tag.key) && (!normalizedQuery || tag.key.includes(normalizedQuery))
  ));
}

export function addTagSelection(selectedTags, name) {
  const tags = normalizeTagSelection(selectedTags);
  const normalizedName = String(name ?? '').trim();
  const key = canonicalTag(normalizedName);
  if (!key || tags.some((tag) => canonicalTag(tag) === key)) return tags;
  return [...tags, normalizedName];
}

export function removeTagSelection(selectedTags, name) {
  const key = canonicalTag(name);
  return normalizeTagSelection(selectedTags).filter((tag) => canonicalTag(tag) !== key);
}
