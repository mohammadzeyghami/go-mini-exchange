"use client";

// Paper-trading identity: alice or bob, kept in localStorage and sent as the
// X-User header. No auth by design — see README scope.

import {
  createContext,
  useContext,
  useEffect,
  useState,
  type ReactNode,
} from "react";
import { useQueryClient } from "@tanstack/react-query";
import { setApiUser } from "./api";

const UserContext = createContext<{
  user: string;
  setUser: (u: string) => void;
}>({ user: "alice", setUser: () => {} });

export const useUser = () => useContext(UserContext);

export function UserProvider({ children }: { children: ReactNode }) {
  const [user, setUserState] = useState("alice");
  const queryClient = useQueryClient();

  useEffect(() => {
    try {
      const saved = localStorage.getItem("mini-exchange-user");
      if (saved) {
        setUserState(saved);
        setApiUser(saved);
      }
    } catch {}
  }, []);

  const setUser = (u: string) => {
    setUserState(u);
    setApiUser(u);
    try {
      localStorage.setItem("mini-exchange-user", u);
    } catch {}
    queryClient.invalidateQueries();
  };

  return (
    <UserContext.Provider value={{ user, setUser }}>
      {children}
    </UserContext.Provider>
  );
}
