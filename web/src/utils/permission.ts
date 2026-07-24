import type { User } from "@/types";

export function isAdminUser(user: User | null | undefined): boolean {
  if (!user || !Array.isArray(user.roles)) return false;
  return user.roles.some((role) => Boolean(role?.is_admin) || (typeof role?.name === "string" && role.name.toUpperCase() === "ADMIN"));
}

export function isBannedUser(user: User | null | undefined): boolean {
  if (!user || !Array.isArray(user.roles)) return false;
  return user.roles.some((role) => Boolean(role?.is_banned) || (typeof role?.name === "string" && role.name.toUpperCase() === "BANNED"));
}

export function hasPermission(
  user: User | null | undefined,
  permissionKey: string,
  libraryId?: string,
  guestPermissions?: string[]
): boolean {
  if (typeof permissionKey !== "string" || !permissionKey) return false;

  if (!user || !Array.isArray(user.roles) || user.roles.length === 0) {
    if (!user && Array.isArray(guestPermissions)) {
      return guestPermissions.includes(permissionKey);
    }
    return false;
  }

  if (isBannedUser(user)) {
    return false;
  }

  if (isAdminUser(user)) {
    return true;
  }

  const sortedRoles = user.roles.filter(Boolean).sort((a, b) => {
    const posA = a.position ?? 0;
    const posB = b.position ?? 0;
    return posB - posA;
  });

  let allowed = false;
  for (const role of sortedRoles) {
    if (Array.isArray(role.permissions)) {
      for (const p of role.permissions) {
        if (!p || p.permission_key !== permissionKey) continue;
        if (p.effect !== "allow" && p.effect !== "deny") return false;

        const conditions = p.conditions;
        if (conditions !== undefined) {
          if (!conditions || typeof conditions !== "object" || Array.isArray(conditions)) return false;
          const keys = Object.keys(conditions);
          if (keys.some((key) => key !== "library_ids")) return false;
          if ("library_ids" in conditions) {
            const libraryIds = conditions.library_ids;
            if (!Array.isArray(libraryIds) || libraryIds.some((id) => typeof id !== "string")) return false;
            if (libraryIds.length > 0 && (!libraryId || !libraryIds.includes(libraryId))) continue;
          }
        }

        if (p.effect === "deny") {
          return false;
        }

        allowed = true;
      }
    }
  }

  return allowed;
}
