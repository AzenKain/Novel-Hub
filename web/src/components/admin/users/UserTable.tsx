import { KeyRound, RotateCcw, Shield, Trash2, UserCog } from "lucide-react";
import React from "react";

import type { User } from "@/types";

type UserTableProps = {
  users: User[];
  t: (key: string, fallback: string) => string;
  onEdit: (user: User) => void;
  onPassword: (user: User) => void;
  onRoles: (user: User) => void;
  onDelete: (user: User) => void;
  onRestore: (user: User) => void;
  currentUserId?: string;
  isCallerOwner?: boolean;
};

export const UserTable: React.FC<UserTableProps> = ({
  users,
  t,
  onEdit,
  onPassword,
  onRoles,
  onDelete,
  onRestore,
  currentUserId,
  isCallerOwner = false,
}) => (
  <div className="card overflow-hidden border border-base-200 bg-base-100 shadow-sm">
    <div className="overflow-x-auto">
      <table className="table">
        <thead>
          <tr className="bg-base-200/50">
            <th className="text-xs font-semibold uppercase tracking-wider opacity-70">
              {t("admin.users", "User")}
            </th>
            <th className="text-xs font-semibold uppercase tracking-wider opacity-70">
              {t("admin.role", "Roles")}
            </th>
            <th className="text-xs font-semibold uppercase tracking-wider opacity-70">
              {t("admin.status", "Status")}
            </th>
            <th className="text-xs font-semibold uppercase tracking-wider opacity-70">
              {t("admin.joined", "Joined")}
            </th>
            <th className="text-right text-xs font-semibold uppercase tracking-wider opacity-70">
              {t("admin.actions", "Actions")}
            </th>
          </tr>
        </thead>
        <tbody>
          {users.length === 0 ? (
            <tr>
              <td colSpan={5} className="py-12 text-center opacity-50">
                {t("admin.no_users", "No users found. Try adjusting your search.")}
              </td>
            </tr>
          ) : (
            users.map((item) => {
              const isOwner = Boolean(item.is_owner);
              const isSelf = item.id === currentUserId;
              const isAdminUser = item.roles?.some(r => Boolean(r.is_admin) || r.name?.toUpperCase() === "ADMIN");

              const canDelete = !isOwner && !isSelf && (!isAdminUser || isCallerOwner);
              const canManage = isCallerOwner || isSelf || (!isAdminUser && !isOwner);

              return (
                <tr
                  key={item.id}
                  className={`hover ${item.is_deleted ? "bg-base-200 opacity-60" : ""}`}
                >
                  <td>
                    <div className="flex items-center gap-4">
                      <div className="avatar placeholder">
                        <div className="flex h-8 w-8 items-center justify-center rounded-full bg-primary/10 text-primary">
                          <span className="text-sm font-bold uppercase">
                            {item.full_name?.charAt(0) || item.email.charAt(0)}
                          </span>
                        </div>
                      </div>
                      <div>
                        <div className="font-bold">
                          {item.full_name || "Unnamed"}
                        </div>
                        <div className="text-sm opacity-60">{item.email}</div>
                      </div>
                    </div>
                  </td>
                  <td>
                    <div className="flex flex-wrap gap-1">
                      {item.roles.map((role) => (
                        <span
                          key={role.id}
                          className="badge badge-primary badge-outline badge-sm font-semibold"
                        >
                          {role.name}
                        </span>
                      ))}
                      {item.roles.length === 0 && (
                        <span className="text-sm opacity-50">No role</span>
                      )}
                    </div>
                  </td>
                  <td>
                    <span
                      className={`badge badge-sm font-medium ${
                        item.is_deleted
                          ? "badge-error badge-outline"
                          : "badge-success badge-outline"
                      }`}
                    >
                      {item.is_deleted
                        ? t("admin.deleted", "Deleted")
                        : t("admin.active", "Active")}
                    </span>
                  </td>
                  <td className="text-sm opacity-70">
                    {item.created_at
                      ? new Date(item.created_at).toLocaleDateString()
                      : "-"}
                  </td>
                  <td className="text-right">
                    <div className="join justify-end">
                      {!item.is_deleted ? (
                        <>
                          {canManage && (
                            <button
                              onClick={() => onEdit(item)}
                              className="btn btn-ghost join-item btn-sm"
                              title="Edit user"
                            >
                              <UserCog className="h-4 w-4" />
                            </button>
                          )}
                          {canManage && (
                            <button
                              onClick={() => onPassword(item)}
                              className="btn btn-ghost join-item btn-sm text-warning hover:bg-warning/10"
                              title="Change password"
                            >
                              <KeyRound className="h-4 w-4" />
                            </button>
                          )}
                          {canManage && (
                            <button
                              onClick={() => onRoles(item)}
                              className="btn btn-ghost join-item btn-sm text-success hover:bg-success/10"
                              title="Manage roles"
                            >
                              <Shield className="h-4 w-4" />
                            </button>
                          )}
                          {canDelete && (
                            <button
                              onClick={() => onDelete(item)}
                              className="btn btn-ghost join-item btn-sm text-error hover:bg-error/10"
                              title="Delete user"
                            >
                              <Trash2 className="h-4 w-4" />
                            </button>
                          )}
                        </>
                      ) : (
                        <>
                          {isCallerOwner && (
                            <button
                              onClick={() => onRestore(item)}
                              className="btn btn-ghost join-item btn-sm text-success hover:bg-success/10"
                              title="Restore user"
                            >
                              <RotateCcw className="h-4 w-4" />
                            </button>
                          )}
                        </>
                      )}
                    </div>
                  </td>
                </tr>
              );
            })
          )}
        </tbody>
      </table>
    </div>
  </div>
);
