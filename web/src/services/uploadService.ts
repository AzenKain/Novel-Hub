import { api } from "../config/api";
import { CommonResponse } from "@/types";

const CHUNK_SIZE = 10 * 1024 * 1024;

export interface UploadCommitParams {
  target: "library" | "book";
  library_id?: string;
  book_id?: string;
  filename: string;
  total_chunks: number;
}

export const uploadService = {
  uploadFileChunked: async (
    file: File,
    target: "library" | "book",
    id: string,
    onProgress?: (progress: number) => void
  ): Promise<CommonResponse<any>> => {
    const initRes = await api.post<CommonResponse<any>>("/upload/init");
    if (!initRes.data.status || !initRes.data.data?.upload_id) {
      throw new Error("Failed to initialize upload");
    }
    const uploadId = initRes.data.data.upload_id;

    const totalChunks = Math.ceil(file.size / CHUNK_SIZE);
    
    for (let chunkIndex = 0; chunkIndex < totalChunks; chunkIndex++) {
      const start = chunkIndex * CHUNK_SIZE;
      const end = Math.min(start + CHUNK_SIZE, file.size);
      const chunk = file.slice(start, end);

      const formData = new FormData();
      formData.append("file", chunk, file.name);
      formData.append("chunk_index", chunkIndex.toString());
      formData.append("total_chunks", totalChunks.toString());

      const res = await api.post<CommonResponse<any>>(`/upload/${uploadId}/chunk`, formData, {
        headers: {
          "Content-Type": "multipart/form-data",
        }
      });

      if (!res.data.status) {
        throw new Error(`Failed to upload chunk ${chunkIndex}`);
      }

      if (onProgress) {
        onProgress(Math.round(((chunkIndex + 1) / totalChunks) * 100));
      }
    }

    const commitParams: UploadCommitParams = {
      target,
      filename: file.name,
      total_chunks: totalChunks,
    };
    
    if (target === "library") {
      commitParams.library_id = id;
    } else {
      commitParams.book_id = id;
    }

    const commitRes = await api.post<CommonResponse<any>>(`/upload/${uploadId}/commit`, commitParams);
    return commitRes.data;
  }
};
