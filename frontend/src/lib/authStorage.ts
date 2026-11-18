const key = "minioa.token";
export const authStorage = {
  get: () => localStorage.getItem(key),
  set: (token: string) => localStorage.setItem(key, token),
  clear: () => localStorage.removeItem(key),
};
