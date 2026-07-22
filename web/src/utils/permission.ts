import type { User } from "@/types";

export function isAdminUser(user: User | null | undefined): boolean {
  if (!user || !user.roles) return false;
  return user.roles.some((role) => Boolean(role.is_admin) || role.name.toUpperCase() === "ADMIN");
}

export function isBannedUser(user: User | null | undefined): boolean {
  if (!user || !user.roles) return false;
  return user.roles.some((role) => Boolean(role.is_banned) || role.name?.toUpperCase() === "BANNED");
}

export function hasPermission(
  user: User | null | undefined,
  permissionKey: string,
  libraryId?: string,
  guestPermissions?: string[]
): boolean {
  if (!user || !user.roles || user.roles.length === 0) {
    if (guestPermissions && Array.isArray(guestPermissions)) {
      return guestPermissions.includes(permissionKey);
    }
    return false;
  }

  if (isAdminUser(user)) {
    return true;
  }

  if (isBannedUser(user)) {
    return false;
  }

  const sortedRoles = [...user.roles].sort((a, b) => {
    const posA = a.position ?? 0;
    const posB = b.position ?? 0;
    return posB - posA;
  });

  let allowed = false;
  for (const role of sortedRoles) {
    if (role.permissions && role.permissions.length > 0) {
      for (const p of role.permissions) {
        if (p.permission_key !== permissionKey) continue;

        if (p.conditions && Array.isArray(p.conditions.library_ids) && p.conditions.library_ids.length > 0) {
          if (!libraryId || !(p.conditions.library_ids as string[]).includes(libraryId)) {
            continue;
          }
        }

        if (p.effect === "deny") {
          return false;
        }

        if (p.effect === "allow") {
          allowed = true;
        }
      }
    }
  }

  return allowed;
}
