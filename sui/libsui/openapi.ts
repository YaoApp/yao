/**
 * SUI OpenAPI Client
 *
 * A lightweight HTTP client for Yao OpenAPI endpoints.
 * Adapted from CUI OpenAPI for use in SUI (browser-only, no build tools).
 *
 * Features:
 * - RESTful API methods (GET, POST, PUT, DELETE, Upload)
 * - Secure cookie authentication
 * - CSRF protection
 * - File upload with progress tracking
 * - Cross-origin support
 *
 * Usage:
 *   const api = new OpenAPI({ baseURL: '/api' })
 *   const response = await api.Get('/users')
 *   if (api.IsError(response)) {
 *     console.error(response.error)
 *   } else {
 *     console.log(response.data)
 *   }
 */

// ============================================================================
// Types
// ============================================================================

interface OpenAPIConfig {
  baseURL: string;
  timeout?: number;
  defaultHeaders?: Record<string, string>;
}

interface ErrorResponse {
  error: string;
  error_description?: string;
  error_uri?: string;
  [key: string]: any;
}

interface ApiResponse<T = any> {
  data?: T;
  error?: ErrorResponse;
  status: number;
  headers: Headers;
}

interface FileUploadOptions {
  uploaderID?: string;
  originalFilename?: string;
  groups?: string[];
  gzip?: boolean;
  compressImage?: boolean;
  compressSize?: number;
  path?: string;
  chunked?: boolean;
  chunkSize?: number;
  public?: boolean;
  share?: "private" | "team";
}

interface FileListOptions {
  uploaderID?: string;
  page?: number;
  pageSize?: number;
  status?: string;
  contentType?: string;
  name?: string;
  orderBy?: string;
  select?: string[];
}

interface FileInfo {
  file_id: string;
  user_path: string;
  path: string;
  bytes: number;
  created_at: number;
  filename: string;
  content_type: string;
  status: string;
  url?: string;
  metadata?: Record<string, any>;
  uploader?: string;
  groups?: string[];
  public?: boolean;
  share?: "private" | "team";
}

interface FileListResponse {
  data: FileInfo[];
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
}

interface FileExistsResponse {
  exists: boolean;
  fileId: string;
}

interface FileDeleteResponse {
  message: string;
  fileId: string;
}

type UploadProgressCallback = (progress: {
  loaded: number;
  total: number;
  percentage: number;
}) => void;

// ============================================================================
// OpenAPI Client
// ============================================================================

class OpenAPI {
  private config: OpenAPIConfig;

  constructor(config: OpenAPIConfig) {
    this.config = config;
  }

  private async handleResponse<T>(response: Response): Promise<ApiResponse<T>> {
    const apiResponse: ApiResponse<T> = {
      status: response.status,
      headers: response.headers,
    };

    try {
      const contentType = response.headers.get("content-type") || "";

      if (contentType.includes("application/json")) {
        const jsonData = await response.json();

        if (!response.ok && jsonData.error) {
          apiResponse.error = jsonData as ErrorResponse;
        } else if (response.ok) {
          apiResponse.data = jsonData as T;
        } else {
          apiResponse.error = {
            error: "http_error",
            error_description: `HTTP ${response.status}: ${response.statusText}`,
          };
        }
      } else if (response.ok) {
        const textData = await response.text();
        apiResponse.data = textData as unknown as T;
      } else {
        const errorText = await response.text();
        apiResponse.error = {
          error: "http_error",
          error_description:
            errorText || `HTTP ${response.status}: ${response.statusText}`,
        };
      }
    } catch (parseError) {
      apiResponse.error = {
        error: "parse_error",
        error_description: `Failed to parse response: ${
          parseError instanceof Error ? parseError.message : "Unknown error"
        }`,
      };
    }

    return apiResponse;
  }

  async Get<T = any>(
    path: string,
    query: Record<string, string> = {},
    headersInit: Record<string, string> = {}
  ): Promise<ApiResponse<T>> {
    const headers = { "Content-Type": "application/json", ...headersInit };
    this.addCSRFToken(headers);

    const queryString = new URLSearchParams(query).toString();
    let url = `${this.config.baseURL}${path}`;
    if (queryString) {
      url += path.includes("?") ? `&${queryString}` : `?${queryString}`;
    }

    const response = await fetch(url, {
      method: "GET",
      headers,
      credentials: "include",
    });

    return this.handleResponse<T>(response);
  }

  async Post<T = any>(
    path: string,
    payload: any,
    headersInit: Record<string, string> = {}
  ): Promise<ApiResponse<T>> {
    const headers = { "Content-Type": "application/json", ...headersInit };
    this.addCSRFToken(headers);

    const response = await fetch(`${this.config.baseURL}${path}`, {
      method: "POST",
      body: typeof payload === "object" ? JSON.stringify(payload) : payload,
      headers,
      credentials: "include",
    });

    return this.handleResponse<T>(response);
  }

  async Put<T = any>(
    path: string,
    payload: any,
    headersInit: Record<string, string> = {}
  ): Promise<ApiResponse<T>> {
    const headers = { "Content-Type": "application/json", ...headersInit };
    this.addCSRFToken(headers);

    const response = await fetch(`${this.config.baseURL}${path}`, {
      method: "PUT",
      body: typeof payload === "object" ? JSON.stringify(payload) : payload,
      headers,
      credentials: "include",
    });

    return this.handleResponse<T>(response);
  }

  async Delete<T = any>(
    path: string,
    headersInit: Record<string, string> = {},
    payload?: any
  ): Promise<ApiResponse<T>> {
    const headers = { "Content-Type": "application/json", ...headersInit };
    this.addCSRFToken(headers);

    const requestOptions: RequestInit = {
      method: "DELETE",
      headers,
      credentials: "include",
    };

    if (payload !== undefined) {
      requestOptions.body = JSON.stringify(payload);
    }

    const response = await fetch(
      `${this.config.baseURL}${path}`,
      requestOptions
    );

    return this.handleResponse<T>(response);
  }

  async Upload<T = any>(
    path: string,
    formData: FormData,
    headersInit: Record<string, string> = {}
  ): Promise<ApiResponse<T>> {
    const headers = { ...headersInit };
    this.addCSRFToken(headers);
    // Don't set Content-Type for FormData - browser sets it with boundary

    const response = await fetch(`${this.config.baseURL}${path}`, {
      method: "POST",
      body: formData,
      headers,
      credentials: "include",
    });

    return this.handleResponse<T>(response);
  }

  // ============================================================================
  // Helper Methods
  // ============================================================================

  IsError<T>(
    response: ApiResponse<T>
  ): response is ApiResponse<T> & { error: ErrorResponse } {
    return response.error !== undefined;
  }

  GetData<T>(response: ApiResponse<T>): T | null {
    return response.data || null;
  }

  SetCSRFToken(token: string): void {
    if (typeof localStorage !== "undefined") {
      localStorage.setItem("csrf_token", token);
    }
  }

  ClearTokens(): void {
    if (typeof localStorage !== "undefined") {
      localStorage.removeItem("csrf_token");
      localStorage.removeItem("xsrf_token");
    }
  }

  IsCrossOrigin(): boolean {
    if (typeof window === "undefined") {
      return false;
    }

    try {
      const apiUrl = new URL(this.config.baseURL, window.location.origin);
      return apiUrl.origin !== window.location.origin;
    } catch {
      return true;
    }
  }

  getBaseURL(): string {
    return this.config.baseURL;
  }

  // ============================================================================
  // Private Methods
  // ============================================================================

  private addCSRFToken(headers: Record<string, string>): void {
    // Try cookies
    const cookieToken =
      this.getSecureCookie("__Host-csrf_token") ||
      this.getSecureCookie("__Secure-csrf_token") ||
      this.getSecureCookie("__Host-xsrf_token") ||
      this.getSecureCookie("__Secure-xsrf_token");

    if (cookieToken) {
      headers["X-CSRF-Token"] = cookieToken;
      return;
    }

    // Try localStorage
    if (typeof localStorage !== "undefined") {
      const storedToken =
        localStorage.getItem("csrf_token") ||
        localStorage.getItem("xsrf_token");
      if (storedToken) {
        headers["X-CSRF-Token"] = storedToken;
        return;
      }
    }

    // Try meta tag
    if (typeof document !== "undefined") {
      const metaToken =
        document
          .querySelector('meta[name="csrf-token"]')
          ?.getAttribute("content") ||
        document
          .querySelector('meta[name="xsrf-token"]')
          ?.getAttribute("content");
      if (metaToken) {
        headers["X-CSRF-Token"] = metaToken;
      }
    }
  }

  private getSecureCookie(name: string): string | null {
    if (typeof document === "undefined") {
      return null;
    }

    const value = `; ${document.cookie}`;
    const parts = value.split(`; ${name}=`);

    if (parts.length === 2) {
      const cookieValue = parts.pop()?.split(";").shift();
      return cookieValue ? decodeURIComponent(cookieValue) : null;
    }

    return null;
  }
}

// ============================================================================
// File API
// ============================================================================

class FileAPI {
  private api: OpenAPI;
  private defaultUploader: string;

  constructor(api: OpenAPI, defaultUploader?: string) {
    this.api = api;
    this.defaultUploader = defaultUploader || "__yao.attachment";
  }

  async Upload(
    file: File,
    options: FileUploadOptions = {},
    onProgress?: UploadProgressCallback
  ): Promise<ApiResponse<FileInfo>> {
    const uploaderID = options.uploaderID || this.defaultUploader;

    const shouldUseChunked =
      options.chunked || file.size > (options.chunkSize || 2 * 1024 * 1024);

    if (shouldUseChunked) {
      return this.uploadChunked(uploaderID, file, options, onProgress);
    }

    const formData = new FormData();
    formData.append("file", file);

    if (options.originalFilename || file.name) {
      formData.append(
        "original_filename",
        options.originalFilename || file.name
      );
    }
    if (options.path) formData.append("path", options.path);
    if (options.groups?.length)
      formData.append("groups", options.groups.join(","));
    if (options.gzip) formData.append("gzip", "true");
    if (options.compressImage) formData.append("compress_image", "true");
    if (options.compressSize)
      formData.append("compress_size", options.compressSize.toString());
    if (options.public !== undefined)
      formData.append("public", options.public ? "true" : "false");
    if (options.share) formData.append("share", options.share);

    if (onProgress) {
      return this.uploadWithProgress(uploaderID, formData, onProgress);
    }

    return this.api.Upload<FileInfo>(`/file/${uploaderID}`, formData);
  }

  async UploadMultiple(
    files: File[],
    options: FileUploadOptions = {},
    onProgress?: (
      fileIndex: number,
      progress: { loaded: number; total: number; percentage: number }
    ) => void
  ): Promise<ApiResponse<FileInfo>[]> {
    const uploadPromises = files.map((file, index) => {
      const progressCallback = onProgress
        ? (progress: { loaded: number; total: number; percentage: number }) =>
            onProgress(index, progress)
        : undefined;
      return this.Upload(file, options, progressCallback);
    });

    return Promise.all(uploadPromises);
  }

  async List(
    options: FileListOptions = {}
  ): Promise<ApiResponse<FileListResponse>> {
    const uploaderID = options.uploaderID || this.defaultUploader;
    const params: Record<string, string> = {};

    if (options.page) params.page = options.page.toString();
    if (options.pageSize) params.page_size = options.pageSize.toString();
    if (options.status) params.status = options.status;
    if (options.contentType) params.content_type = options.contentType;
    if (options.name) params.name = options.name;
    if (options.orderBy) params.order_by = options.orderBy;
    if (options.select?.length) params.select = options.select.join(",");

    return this.api.Get<FileListResponse>(`/file/${uploaderID}`, params);
  }

  async Retrieve(
    fileID: string,
    uploaderID?: string
  ): Promise<ApiResponse<FileInfo>> {
    if (!fileID) throw new Error("File ID is required");
    const actualUploaderID = uploaderID || this.defaultUploader;
    return this.api.Get<FileInfo>(
      `/file/${actualUploaderID}/${encodeURIComponent(fileID)}`
    );
  }

  async Delete(
    fileID: string,
    uploaderID?: string
  ): Promise<ApiResponse<FileDeleteResponse>> {
    if (!fileID) throw new Error("File ID is required");
    const actualUploaderID = uploaderID || this.defaultUploader;
    return this.api.Delete<FileDeleteResponse>(
      `/file/${actualUploaderID}/${encodeURIComponent(fileID)}`
    );
  }

  async Download(
    fileID: string,
    uploaderID?: string
  ): Promise<ApiResponse<Blob>> {
    if (!fileID) throw new Error("File ID is required");
    const actualUploaderID = uploaderID || this.defaultUploader;
    const url = `${this.api.getBaseURL()}/file/${actualUploaderID}/${encodeURIComponent(
      fileID
    )}/content`;

    const response = await fetch(url, {
      method: "GET",
      credentials: "include",
    });

    const blob = await response.blob();
    const apiResponse: ApiResponse<Blob> = {
      data: blob,
      status: response.status,
      headers: response.headers,
    };

    if (!response.ok) {
      apiResponse.error = {
        error: "download_failed",
        error_description: `Download failed with status ${response.status}`,
      };
    }

    return apiResponse;
  }

  async Exists(
    fileID: string,
    uploaderID?: string
  ): Promise<ApiResponse<FileExistsResponse>> {
    if (!fileID) throw new Error("File ID is required");
    const actualUploaderID = uploaderID || this.defaultUploader;
    return this.api.Get<FileExistsResponse>(
      `/file/${actualUploaderID}/${encodeURIComponent(fileID)}/exists`
    );
  }

  // Static utility methods
  static FormatSize(bytes: number): string {
    if (bytes === 0) return "0 Bytes";
    const k = 1024;
    const sizes = ["Bytes", "KB", "MB", "GB", "TB"];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i];
  }

  static GetExtension(filename: string): string {
    return filename.slice(((filename.lastIndexOf(".") - 1) >>> 0) + 2);
  }

  static IsImage(contentType: string): boolean {
    return contentType.startsWith("image/");
  }

  static IsDocument(contentType: string): boolean {
    const documentTypes = [
      "application/pdf",
      "application/msword",
      "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
      "application/vnd.ms-excel",
      "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
      "application/vnd.ms-powerpoint",
      "application/vnd.openxmlformats-officedocument.presentationml.presentation",
      "text/plain",
      "text/csv",
    ];
    return documentTypes.includes(contentType);
  }

  // Private methods
  private async uploadChunked(
    uploaderID: string,
    file: File,
    options: FileUploadOptions = {},
    onProgress?: UploadProgressCallback
  ): Promise<ApiResponse<FileInfo>> {
    const chunkSize = options.chunkSize || 2 * 1024 * 1024;
    const totalSize = file.size;
    const totalChunks = Math.ceil(totalSize / chunkSize);
    const fileUID = this.generateUID();

    let lastResponse: ApiResponse<FileInfo> | null = null;

    for (let chunkIndex = 0; chunkIndex < totalChunks; chunkIndex++) {
      const start = chunkIndex * chunkSize;
      const end = Math.min(start + chunkSize - 1, totalSize - 1);
      const chunkBlob = file.slice(start, end + 1);

      const formData = new FormData();
      formData.append("file", chunkBlob);

      if (chunkIndex === 0) {
        if (options.originalFilename || file.name) {
          formData.append(
            "original_filename",
            options.originalFilename || file.name
          );
        }
        if (options.path) formData.append("path", options.path);
        if (options.groups?.length)
          formData.append("groups", options.groups.join(","));
        if (options.gzip) formData.append("gzip", "true");
        if (options.compressImage) formData.append("compress_image", "true");
        if (options.compressSize)
          formData.append("compress_size", options.compressSize.toString());
        if (options.public !== undefined)
          formData.append("public", options.public ? "true" : "false");
        if (options.share) formData.append("share", options.share);
      }

      const chunkResponse = await this.uploadChunk(
        uploaderID,
        formData,
        start,
        end,
        totalSize,
        fileUID
      );

      if (this.api.IsError(chunkResponse)) {
        return chunkResponse;
      }

      lastResponse = chunkResponse;

      if (onProgress) {
        const loaded = end + 1;
        const percentage = Math.round((loaded / totalSize) * 100);
        onProgress({ loaded, total: totalSize, percentage });
      }
    }

    return lastResponse!;
  }

  private uploadChunk(
    uploaderID: string,
    formData: FormData,
    start: number,
    end: number,
    total: number,
    uid: string
  ): Promise<ApiResponse<FileInfo>> {
    return new Promise((resolve) => {
      const xhr = new XMLHttpRequest();

      xhr.addEventListener("load", () => {
        try {
          const response = JSON.parse(xhr.responseText);
          const apiResponse: ApiResponse<FileInfo> = {
            data: response.data || response,
            status: xhr.status,
            headers: new Headers(),
          };

          if (xhr.status >= 200 && xhr.status < 300) {
            resolve(apiResponse);
          } else {
            apiResponse.error = response.error || {
              error: "chunk_upload_failed",
              error_description: `Chunk upload failed with status ${xhr.status}`,
            };
            resolve(apiResponse);
          }
        } catch {
          resolve({
            status: xhr.status,
            headers: new Headers(),
            error: {
              error: "parse_error",
              error_description: "Failed to parse chunk response",
            },
          });
        }
      });

      xhr.addEventListener("error", () => {
        resolve({
          status: xhr.status || 0,
          headers: new Headers(),
          error: {
            error: "network_error",
            error_description: "Network error during chunk upload",
          },
        });
      });

      xhr.open("POST", `${this.api.getBaseURL()}/file/${uploaderID}`);
      xhr.setRequestHeader("Content-Sync", "true");
      xhr.setRequestHeader("Content-Uid", uid);
      xhr.setRequestHeader("Content-Range", `bytes ${start}-${end}/${total}`);

      const csrfToken = this.getCSRFToken();
      if (csrfToken) {
        xhr.setRequestHeader("X-CSRF-Token", csrfToken);
      }

      xhr.withCredentials = true;
      xhr.send(formData);
    });
  }

  private uploadWithProgress(
    uploaderID: string,
    formData: FormData,
    onProgress: UploadProgressCallback
  ): Promise<ApiResponse<FileInfo>> {
    return new Promise((resolve) => {
      const xhr = new XMLHttpRequest();

      xhr.upload.addEventListener("progress", (event) => {
        if (event.lengthComputable) {
          const percentage = Math.round((event.loaded / event.total) * 100);
          onProgress({ loaded: event.loaded, total: event.total, percentage });
        }
      });

      xhr.addEventListener("load", () => {
        try {
          const response = JSON.parse(xhr.responseText);
          const apiResponse: ApiResponse<FileInfo> = {
            data: response.data || response,
            status: xhr.status,
            headers: new Headers(),
          };

          if (xhr.status >= 200 && xhr.status < 300) {
            resolve(apiResponse);
          } else {
            apiResponse.error = response.error || {
              error: "upload_failed",
              error_description: `Upload failed with status ${xhr.status}`,
            };
            resolve(apiResponse);
          }
        } catch {
          resolve({
            status: xhr.status,
            headers: new Headers(),
            error: {
              error: "parse_error",
              error_description: "Failed to parse response",
            },
          });
        }
      });

      xhr.addEventListener("error", () => {
        resolve({
          status: xhr.status || 0,
          headers: new Headers(),
          error: {
            error: "network_error",
            error_description: "Network error during upload",
          },
        });
      });

      xhr.open("POST", `${this.api.getBaseURL()}/file/${uploaderID}`);

      const csrfToken = this.getCSRFToken();
      if (csrfToken) {
        xhr.setRequestHeader("X-CSRF-Token", csrfToken);
      }

      xhr.withCredentials = true;
      xhr.send(formData);
    });
  }

  private getCSRFToken(): string | null {
    if (typeof document !== "undefined") {
      const cookies = document.cookie.split(";");
      for (const cookie of cookies) {
        const [name, value] = cookie.trim().split("=");
        if (
          name === "__Host-csrf_token" ||
          name === "__Secure-csrf_token" ||
          name === "__Host-xsrf_token" ||
          name === "__Secure-xsrf_token"
        ) {
          return decodeURIComponent(value);
        }
      }
    }

    if (typeof localStorage !== "undefined") {
      return (
        localStorage.getItem("csrf_token") || localStorage.getItem("xsrf_token")
      );
    }

    return null;
  }

  private generateUID(): string {
    // Simple unique ID generator (no external dependencies)
    return "xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx".replace(/[xy]/g, (c) => {
      const r = (Math.random() * 16) | 0;
      const v = c === "x" ? r : (r & 0x3) | 0x8;
      return v.toString(16);
    });
  }
}

// ============================================================================
// Global Registration for SUI
// ============================================================================

// Make available globally for SUI pages (no export, direct global assignment)
(window as any).OpenAPI = OpenAPI;
(window as any).FileAPI = FileAPI;
