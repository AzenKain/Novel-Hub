import { api } from "@/config/api";
import type {
  CommonResponse,
  ValidationReport,
  BookRepairResult,
  RepairOptions,
} from "@/types";

export const bookDoctorService = {
  async validateBook(
    bookId: string,
    fileId?: string,
  ): Promise<ValidationReport> {
    const params = fileId ? { file_id: fileId } : {};
    const res = await api.get<CommonResponse<ValidationReport>>(
      `/books/${encodeURIComponent(bookId)}/doctor/validate`,
      { params },
    );
    return res.data.data!;
  },

  async repairBook(
    bookId: string,
    options?: RepairOptions,
    fileId?: string,
  ): Promise<BookRepairResult> {
    const params = fileId ? { file_id: fileId } : {};
    const res = await api.post<CommonResponse<BookRepairResult>>(
      `/books/${encodeURIComponent(bookId)}/doctor/repair`,
      options || {},
      { params },
    );
    return res.data.data!;
  },
};
