import { writable } from "svelte/store";

export type User = {
  id: string;
  username: string;
  avatarUrl?: string;
  email: string;
  // token: string;
};

// There is no authentication yet: every request identifies itself as this
// placeholder, the same one boards are created with.
const user = writable<User | null>({ id: 'Messi', username: 'Lionel Messi', email: 'messi@example.com' });

const logout = () => {
  user.set(null);
  localStorage.removeItem('user');
};

const getUser = { subscribe: user.subscribe };

const setUser = (newUser: User) => {
  user.set(newUser);
  localStorage.setItem('user', JSON.stringify(newUser));
};

const loadUserFromLocalStorage = () => {
  const storedUser = localStorage.getItem('user');
  if (storedUser) {
    user.set(JSON.parse(storedUser));
  }
};

loadUserFromLocalStorage();

export { logout, getUser, setUser };