// Validate useful public business results without retaining raw response data.
const nonempty = value => typeof value === 'string' && value.trim().length > 0;
const store = value => value && Number.isSafeInteger(value.id) && value.id > 0 &&
  nonempty(value.name) && typeof value.address === 'string' && Number.isInteger(value.wait);
const date = value => {
  if (!/^\d{8}$/.test(value)) return false;
  const [year, month, day] = [Number(value.slice(0, 4)), Number(value.slice(4, 6)), Number(value.slice(6, 8))];
  const parsed = new Date(Date.UTC(year, month - 1, day));
  return parsed.getUTCFullYear() === year && parsed.getUTCMonth() === month - 1 && parsed.getUTCDate() === day;
};
const time = value => /^(?:[01]\d|2[0-3])[0-5]\d[0-5]\d$/.test(value);
export function publicShape(label, data) {
  if (!Array.isArray(data) || data.length === 0) return false;
  if (label === 'stores') return data.length <= 2 && data.every(store);
  if (label === 'store-3006') return data.length === 1 && store(data[0]) && data[0].id === 3006;
  if (label === 'slots-3006-2-T') return data.every(value => value &&
    ['storeId', 'date', 'start', 'end', 'availability'].every(key => nonempty(value[key])) && value.storeId === '3006' &&
    date(value.date) && time(value.start) && time(value.end));
  return false;
}
