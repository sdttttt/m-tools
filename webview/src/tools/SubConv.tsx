import { createSignal, For, Show } from "solid-js";
import { SubConv, type SubconvConvertResult } from "~/mygo";
import { MotionButton } from "~/components/motion";
import { Card, CardContent } from "~/components/ui/card";
import { Label } from "~/components/ui/label";
import { TextField, TextFieldInput } from "~/components/ui/text-field";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "~/components/ui/select";

const FROM_LABELS: Record<string, string> = {
  auto: "自动识别",
  v2ray: "v2ray 订阅",
  clash: "Clash 订阅",
  singbox: "sing-box 订阅",
};
const TO_LABELS: Record<string, string> = {
  v2ray: "v2ray 订阅",
  clash: "Clash 订阅 (YAML)",
  singbox: "sing-box 订阅 (JSON)",
};
const DEFAULT_NAMES: Record<string, string> = {
  v2ray: "subscription.txt",
  clash: "subscription.yaml",
  singbox: "subscription.json",
};

const PROTOCOL_LABELS: Record<string, string> = {
  vless: "VLESS",
  vmess: "VMess",
  trojan: "Trojan",
  shadowsocks: "Shadowsocks",
  hysteria2: "Hysteria2",
  tuic: "TUIC",
  anytls: "AnyTLS",
};

export default function SubConvTool() {
  const [url, setUrl] = createSignal("");
  const [input, setInput] = createSignal("");
  const [from, setFrom] = createSignal("auto");
  const [to, setTo] = createSignal("clash");
  const [busy, setBusy] = createSignal(false);
  const [result, setResult] = createSignal<SubconvConvertResult | null>(null);
  const [error, setError] = createSignal("");
  const [notice, setNotice] = createSignal("");

  const labelOf = (labels: Record<string, string>, id: string) => labels[id] ?? id;
  const idOf = (labels: Record<string, string>, label: string) =>
    Object.entries(labels).find(([, l]) => l === label)?.[0] ?? "";

  async function fetchSub() {
    const u = url().trim();
    if (!u || busy()) return;
    setBusy(true);
    setError("");
    setNotice("");
    try {
      const res = await SubConv.fetch(u);
      setInput(res.body);
      setNotice("已拉取订阅内容，可直接转换");
    } catch (e) {
      setError(String(e).replace(/^Error:\s*/, ""));
    } finally {
      setBusy(false);
    }
  }

  async function convert() {
    if (busy()) return;
    if (!input().trim()) {
      setError("请先粘贴订阅内容或拉取订阅");
      return;
    }
    setBusy(true);
    setError("");
    setNotice("");
    setResult(null);
    try {
      const res = await SubConv.convert({ text: input(), from: from(), to: to() });
      setResult(res);
      setNotice("");
    } catch (e) {
      setError(String(e).replace(/^Error:\s*/, ""));
    } finally {
      setBusy(false);
    }
  }

  async function copyOutput() {
    const r = result();
    if (!r) return;
    try {
      await navigator.clipboard.writeText(r.output);
      setNotice("已复制到剪贴板");
    } catch {
      setError("复制失败，请手动全选复制");
    }
  }

  async function save() {
    const r = result();
    if (!r) return;
    const dest = await SubConv.saveAs(r.output, DEFAULT_NAMES[r.to] ?? "subscription.txt");
    if (dest) setNotice(`已保存到 ${dest}`);
  }

  const statsLine = (r: SubconvConvertResult) => {
    const parts = Object.entries(r.byProtocol)
      .filter(([, n]) => n > 0)
      .map(([p, n]) => `${PROTOCOL_LABELS[p] ?? p} ×${n}`);
    return parts.join("、");
  };

  return (
    <div class="mx-auto flex w-full max-w-3xl flex-col gap-5">
      {/* 输入 */}
      <Card>
        <CardContent class="flex flex-col gap-4 px-5 py-4">
          <div class="flex items-end gap-3">
            <TextField class="flex-1" value={url()} onChange={(v) => setUrl(v)}>
              <Label class="text-xs text-muted-foreground">订阅地址</Label>
              <TextFieldInput
                type="text"
                placeholder="https://example.com/subscribe?token=…"
                autocomplete="off"
                spellcheck={false}
              />
            </TextField>
            <MotionButton variant="outline" onClick={fetchSub} disabled={busy() || !url().trim()}>
              拉取
            </MotionButton>
          </div>
          <div class="flex flex-col gap-1">
            <Label class="text-xs text-muted-foreground">
              订阅内容（粘贴分享链接、base64 订阅包、Clash YAML 或 sing-box JSON）
            </Label>
            <textarea
              class="min-h-36 w-full resize-y rounded-md border border-input bg-transparent px-3 py-2 font-mono text-xs focus:outline-none focus:ring-1 focus:ring-ring"
              value={input()}
              onInput={(e) => setInput(e.currentTarget.value)}
              spellcheck={false}
            />
          </div>
          <div class="flex flex-wrap items-end gap-x-6 gap-y-3">
            <div class="flex flex-col gap-1">
              <Label class="text-xs text-muted-foreground">源格式</Label>
              <Select
                options={Object.values(FROM_LABELS)}
                value={labelOf(FROM_LABELS, from())}
                onChange={(label) => {
                  if (label) setFrom(idOf(FROM_LABELS, label));
                }}
                itemComponent={(props) => (
                  <SelectItem item={props.item}>{props.item.rawValue}</SelectItem>
                )}
              >
                <SelectTrigger class="w-40" aria-label="源格式">
                  <SelectValue<string>>{(state) => state.selectedOption()}</SelectValue>
                </SelectTrigger>
                <SelectContent />
              </Select>
            </div>
            <div class="flex flex-col gap-1">
              <Label class="text-xs text-muted-foreground">目标格式</Label>
              <Select
                options={Object.values(TO_LABELS)}
                value={labelOf(TO_LABELS, to())}
                onChange={(label) => {
                  if (label) setTo(idOf(TO_LABELS, label));
                }}
                itemComponent={(props) => (
                  <SelectItem item={props.item}>{props.item.rawValue}</SelectItem>
                )}
              >
                <SelectTrigger class="w-48" aria-label="目标格式">
                  <SelectValue<string>>{(state) => state.selectedOption()}</SelectValue>
                </SelectTrigger>
                <SelectContent />
              </Select>
            </div>
            <MotionButton onClick={convert} disabled={busy()}>
              转换
            </MotionButton>
          </div>
          <Show when={error()}>
            <p class="m-0 text-xs text-destructive">{error()}</p>
          </Show>
        </CardContent>
      </Card>

      {/* 输出 */}
      <Show when={result()}>
        {(r) => (
          <Card>
            <CardContent class="flex flex-col gap-3 px-5 py-4">
              <div class="flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-muted-foreground">
                <span>
                  识别为 {labelOf(FROM_LABELS, r().from)} · 共 {r().total} 个节点，转出{" "}
                  {r().converted} 个
                </span>
                <Show when={r().skipped.length > 0}>
                  <span class="text-destructive">跳过 {r().skipped.length} 个</span>
                </Show>
                <Show when={statsLine(r())}>
                  <span>{statsLine(r())}</span>
                </Show>
              </div>
              <textarea
                class="min-h-52 w-full resize-y rounded-md border border-input bg-transparent px-3 py-2 font-mono text-xs focus:outline-none focus:ring-1 focus:ring-ring"
                readOnly
                value={r().output}
                spellcheck={false}
              />
              <div class="flex items-center gap-3">
                <MotionButton variant="outline" size="sm" onClick={copyOutput}>
                  复制
                </MotionButton>
                <MotionButton variant="outline" size="sm" onClick={save}>
                  保存到…
                </MotionButton>
                <Show when={notice()}>
                  <span class="truncate text-xs text-muted-foreground">{notice()}</span>
                </Show>
              </div>
              <Show when={r().skipped.length > 0}>
                <div class="flex flex-col gap-1 border-t pt-3">
                  <For each={r().skipped}>
                    {(s) => (
                      <p class="m-0 text-xs">
                        <span class="text-destructive">{s.name}</span>
                        <span class="text-muted-foreground"> — {s.reason}</span>
                      </p>
                    )}
                  </For>
                </div>
              </Show>
            </CardContent>
          </Card>
        )}
      </Show>
    </div>
  );
}
