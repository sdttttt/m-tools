// Shared bits for the image tool pages: label tables, probing helpers and
// the batch-runner loop both tools drive their lists with.

export type ImgFormat = "jpg" | "png" | "gif" | "webp" | "bmp" | "tiff";

export const FORMAT_LABELS: Record<string, string> = {
  jpg: "JPEG（.jpg）",
  png: "PNG（.png）",
  gif: "GIF（.gif）",
  webp: "WebP（.webp）",
  bmp: "BMP（.bmp）",
  tiff: "TIFF（.tiff）",
};

export const QUALITY_LABELS: string[] = ["高画质", "均衡", "紧凑（体积小）"];

export const QUALITY_BY_LABEL: Record<string, "high" | "balanced" | "compact"> = {
  高画质: "high",
  均衡: "balanced",
  "紧凑（体积小）": "compact",
};

export const LABEL_BY_QUALITY: Record<string, string> = {
  high: "高画质",
  balanced: "均衡",
  compact: "紧凑（体积小）",
};

export function fmtSize(n: number): string {
  if (n < 1024) return `${n} B`;
  if (n < 1048576) return `${(n / 1024).toFixed(1)} KB`;
  return `${(n / 1048576).toFixed(1)} MB`;
}

// The page treats tif/tiff and jpg/jpeg as one format: the Go side always
// writes .tiff / .jpg.
export function sourceFormat(path: string, probedFormat?: string): string {
  if (probedFormat) return probedFormat;
  const ext = path.split(".").pop()?.toLowerCase() ?? "";
  if (ext === "jpeg") return "jpg";
  if (ext === "tif") return "tiff";
  return ext;
}

export function fileName(path: string): string {
  return path.split("/").pop() ?? path;
}

export interface ProbeRow {
  path: string;
  format: string;
  width: number;
  height: number;
  sizeBytes: number;
  hasAlpha: boolean;
  supported: boolean;
  skipReason: string;
}

// probeAll probes every picked path; a failed probe becomes an
// unsupported row ("读取失败") instead of rejecting the whole pick.
export async function probeAll(
  probe: (path: string) => Promise<ProbeRow>,
  paths: string[],
): Promise<ProbeRow[]> {
  return Promise.all(
    paths.map(async (p) => {
      try {
        return await probe(p);
      } catch {
        return {
          path: p,
          format: "",
          width: 0,
          height: 0,
          sizeBytes: 0,
          hasAlpha: false,
          supported: false,
          skipReason: "读取失败",
        } as ProbeRow;
      }
    }),
  );
}

export type RowState = "pending" | "converting" | "optimizing" | "done" | "failed" | "skipped";

export const STATE_LABELS: Record<RowState, string> = {
  pending: "待处理",
  converting: "转换中…",
  optimizing: "优化中…",
  done: "完成",
  failed: "失败",
  skipped: "跳过",
};

export const STATE_COLORS: Record<RowState, string> = {
  pending: "text-muted-foreground",
  converting: "text-muted-foreground",
  optimizing: "text-muted-foreground",
  done: "text-primary",
  failed: "text-destructive",
  skipped: "text-destructive",
};
