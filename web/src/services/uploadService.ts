import { api } from "../config/api";
import type { CommonResponse, UploadCommitParams } from "@/types";
import axios from "axios";

const CHUNK_SIZE = 10 * 1024 * 1024;

export type UploadProgressStats = {
  progress: number;
  uploadedBytes: number;
  totalBytes: number;
  speedBytesPerSec: number;
};

export const uploadService = {
  uploadFileChunked: async (
    file: File,
    target: "library" | "book",
    id: string,
    onProgress?: (stats: UploadProgressStats) => void
  ): Promise<CommonResponse<any>> => {
    try {
      const totalChunks = Math.ceil(file.size / CHUNK_SIZE);
      const initPayload: UploadCommitParams & { total_bytes: number } = {
        target,
        filename: file.name,
        total_chunks: totalChunks,
        total_bytes: file.size,
      };
      if (target === "library") initPayload.library_id = id;
      else initPayload.book_id = id;

      const initRes = await api.post<CommonResponse<any>>("/upload/init", initPayload);
      if (!initRes.data.status || !initRes.data.data?.upload_id) {
        throw new Error("Failed to initialize upload");
      }
      const uploadId = initRes.data.data.upload_id;

      const startTime = Date.now();

      for (let chunkIndex = 0; chunkIndex < totalChunks; chunkIndex++) {
        const start = chunkIndex * CHUNK_SIZE;
        const end = Math.min(start + CHUNK_SIZE, file.size);
        const chunkSize = end - start;
        const chunk = file.slice(start, end);

        const formData = new FormData();
        formData.append("file", chunk, file.name);
        formData.append("chunk_index", chunkIndex.toString());
        formData.append("total_chunks", totalChunks.toString());

        const res = await api.post<CommonResponse<any>>(`/upload/${uploadId}/chunk`, formData, {
          headers: {
            "Content-Type": "multipart/form-data",
          },
          onUploadProgress: (progressEvent) => {
            if (!onProgress) return;
            const chunkLoaded = progressEvent.loaded || 0;
            const currentTotalUploaded = start + Math.min(chunkLoaded, chunkSize);
            const elapsedTimeSec = Math.max((Date.now() - startTime) / 1000, 0.1);
            const speed = currentTotalUploaded / elapsedTimeSec;
            const percent = Math.min(Math.round((currentTotalUploaded / file.size) * 100), 99);

            onProgress({
              progress: percent,
              uploadedBytes: currentTotalUploaded,
              totalBytes: file.size,
              speedBytesPerSec: speed,
            });
          },
        });

        if (!res.data.status) {
          throw new Error(`Failed to upload chunk ${chunkIndex}`);
        }
      }

      if (onProgress) {
        const elapsedTimeSec = Math.max((Date.now() - startTime) / 1000, 0.1);
        onProgress({
          progress: 100,
          uploadedBytes: file.size,
          totalBytes: file.size,
          speedBytesPerSec: file.size / elapsedTimeSec,
        });
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
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<any>;
      }
      throw error;
    }
  }
};
