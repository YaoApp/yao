import { Process } from "@yao/runtime";

interface Request {
  query: Record<string, string>;
  params: Record<string, string>;
  locale: string;
  theme: string;
}

const ENV_KEYS = [
  "YAO_ENV",
  "YAO_HOST",
  "YAO_PORT",
  "YAO_SERVER_URL",
  "YAO_OAUTH_ISSUER_URL",
  "YAO_DB_DRIVER",
  "YAO_DB_PRIMARY",
  "YAO_TAI_SERVER",
  "YAO_TAI_TRANSPORT_PORT",
  "YAO_GRPC",
  "TAO_OAUTH_ISSUER",
  "TAO_OAUTH_CLIENT_ID",
  "TAO_OAUTH_LOGIN_REDIRECT_URIS"
];

function maskDSN(value: string): string {
  if (!value) return "";
  return value.replace(/:\/\/([^:/@]+):([^@]+)@/, "://$1:***@");
}

/**
 * GetEnv — expose only non-secret UAT runtime information to the page.
 */
function GetEnv(request: Request): Record<string, string> {
  const raw = Process("utils.env.GetMany", ...ENV_KEYS) as Record<string, string>;
  const taoIssuer = raw["TAO_OAUTH_ISSUER"] || "";
  const taiPort = raw["YAO_TAI_TRANSPORT_PORT"] || "";
  const grpc = raw["YAO_GRPC"] || "on";
  const db = raw["YAO_DB_PRIMARY"] || "";
  const dbHost = db.replace(/^[^@]+@/, "").replace(/\/.*$/, "");
  const clientID = raw["TAO_OAUTH_CLIENT_ID"] || "";
  const shortClient = clientID
    ? clientID.slice(0, 12) + "..." + clientID.slice(-6)
    : "—";

  return {
    ...raw,
    YAO_DB_PRIMARY: maskDSN(db),
    DB_TARGET: dbHost ? dbHost : raw["YAO_DB_DRIVER"] || "",
    TAO_STATUS: taoIssuer ? "Configured" : "Not configured",
    TAO_ISSUER: taoIssuer || "—",
    TAO_CLIENT: shortClient,
    TAI_STATUS: taiPort ? "Enabled" : "Disabled",
    TAI_TRANSPORT: taiPort || "—",
    OPENAPI_GRPC: grpc,
    SERVER: raw["YAO_SERVER_URL"] || "dynamic (request host)",
    OAUTH_ISSUER: raw["YAO_OAUTH_ISSUER_URL"] || "file (openapi.yao)",
    BUILD: "source (yao-dev)",
    APP_ROOT: "yao/e2e/app",
    CUI_URL: "http://127.0.0.1:5199/app/"
  };
}
