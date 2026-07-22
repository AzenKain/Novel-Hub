import type { User } from "@/types";

export function isAdminUser(user: User | null | undefined): boolean {
  if (!user || !user.roles) return false;
  return user.roles.some((role) => Boolean(role.is_admin) || role.name.toUpperCase() === "ADMIN");
}

export function isModOrAdminUser(user: User | null | undefined): boolean {
  if (!user || !user.roles) return false;
  return user.roles.some(
    (role) => Boolean(role.is_admin) || role.name.toUpperCase() === "ADMIN" || role.name.toUpperCase() === "MOD"
  );
}
