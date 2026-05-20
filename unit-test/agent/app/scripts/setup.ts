import { log, Process } from "@yao/runtime";

function Init(force: boolean = false) {
  if (!force && IsInstalled()) {
    log.Info(
      "Application is already initialized. Use 'yao run scripts.setup.Init true' to force reinit."
    );
    return { status: "skipped", message: "Already initialized" };
  }

  log.Info("=== Starting Agent Test Application Init ===");

  log.Info("Running database migration...");
  MigrateModels();

  SetupRoles();
  SetupTypes();
  SetupMenus();
  SetupInvitationCodes();

  const rootInfo = SetupRootUser();

  log.Info("=== Application Init Completed ===");
  console.log("");
  console.log("========================================");
  console.log("  Root User Credentials");
  console.log("----------------------------------------");
  console.log(`  Email:    ${ROOT_USER.email}`);
  console.log(`  Password: ${ROOT_USER_PASSWORD}`);
  if (rootInfo) {
    console.log(`  UserID:   ${rootInfo.userId}`);
    console.log(`  TeamID:   ${rootInfo.teamId}`);
  }
  console.log("========================================");

  return { status: "success", message: "Initialization completed" };
}

function IsInstalled(): boolean {
  try {
    const users = Process("models.__yao.user.Get", {
      wheres: [{ column: "email", value: ROOT_USER.email }],
      limit: 1,
    });
    if (users && users.length > 0) {
      return true;
    }
    const roles = Process("models.__yao.role.Get", { limit: 1 });
    if (roles && roles.length > 0) {
      return true;
    }
    return false;
  } catch (e) {
    return false;
  }
}

function MigrateModels() {
  const models = ["menu"];
  for (const model of models) {
    try {
      Process(`models.${model}.Migrate`, false);
      log.Info(`Migrated model: ${model}`);
    } catch (e) {
      log.Warn(`Failed to migrate model ${model}: ${e}`);
    }
  }
}

// ============================================
// Seed Data Import
// ============================================

function SetupRoles() {
  log.Info("Setting up roles...");
  const result = Process("seeds.import", "roles.csv", "__yao.role", {
    chunk_size: 100,
    duplicate: "ignore",
    mode: "each",
  });
  log.Info(
    `Roles: Total=${result.total}, Success=${result.success}, Ignored=${result.ignore}, Failed=${result.failure}`
  );
}

function SetupTypes() {
  log.Info("Setting up user types...");
  const result = Process("seeds.import", "types.csv", "__yao.user.type", {
    chunk_size: 100,
    duplicate: "ignore",
    mode: "each",
  });
  log.Info(
    `Types: Total=${result.total}, Success=${result.success}, Ignored=${result.ignore}, Failed=${result.failure}`
  );
}

function SetupMenus() {
  log.Info("Setting up menus...");
  const result = Process("seeds.import", "menus.csv", "menu", {
    chunk_size: 100,
    duplicate: "ignore",
    mode: "each",
  });
  log.Info(
    `Menus: Total=${result.total}, Success=${result.success}, Ignored=${result.ignore}, Failed=${result.failure}`
  );
}

function SetupInvitationCodes() {
  log.Info("Setting up invitation codes...");
  const result = Process(
    "seeds.import",
    "invitation_codes.csv",
    "__yao.invitation",
    { chunk_size: 100, duplicate: "ignore", mode: "each" }
  );
  log.Info(
    `Invitation codes: Total=${result.total}, Success=${result.success}, Ignored=${result.ignore}, Failed=${result.failure}`
  );
}

// ============================================
// ID Generation
// ============================================

function generateId(): string {
  const timestamp = Date.now().toString().slice(-8);
  const random = Math.floor(Math.random() * 10000)
    .toString()
    .padStart(4, "0");
  return timestamp + random;
}

// ============================================
// Root User
// ============================================

const ROOT_USER_PASSWORD = "Yao123++";

const ROOT_USER = {
  email: "root@yaoagents.com",
  name: "Administrator",
  status: "active",
  role_id: "system:root",
  type: "selfhosting",
  locale: "en-us",
};

const ROOT_TEAM = {
  name: "Administrator's Team",
  display_name: "Administrator's Team",
  description: "Default team for root user",
  status: "active",
  type: "other",
  is_verified: true,
  role_id: "system:root",
};

function SetupRootUser(): { userId: string; teamId: string } | null {
  log.Info("Setting up root user...");

  const existing = Process("models.__yao.user.Get", {
    wheres: [{ column: "email", value: ROOT_USER.email }],
    limit: 1,
  });

  if (existing && existing.length > 0) {
    log.Info(`Root user already exists: ${ROOT_USER.email}`);
    const userId = existing[0].user_id;
    const teams = Process("models.__yao.team.Get", {
      wheres: [{ column: "owner_id", value: userId }],
      limit: 1,
    });
    const teamId = teams && teams.length > 0 ? teams[0].team_id : null;
    return teamId ? { userId, teamId } : null;
  }

  const userId = generateId();
  const teamId = generateId();
  const memberId = generateId();

  Process("models.__yao.user.Save", {
    ...ROOT_USER,
    user_id: userId,
    password_hash: ROOT_USER_PASSWORD,
  });
  log.Info(`Root user created: ${ROOT_USER.email} (user_id: ${userId})`);

  Process("models.__yao.team.Save", {
    ...ROOT_TEAM,
    team_id: teamId,
    owner_id: userId,
    contact_email: ROOT_USER.email,
  });
  log.Info(`Root team created (team_id: ${teamId})`);

  Process("models.__yao.member.Save", {
    member_id: memberId,
    team_id: teamId,
    user_id: userId,
    member_type: "user",
    display_name: ROOT_USER.name,
    email: ROOT_USER.email,
    role_id: "team:owner",
    is_owner: true,
    status: "active",
    joined_at: new Date().toISOString(),
  });
  log.Info(`Root member created (member_id: ${memberId})`);

  return { userId, teamId };
}

// ============================================
// Test Users
// ============================================

const TEST_PASSWORD = "Test123++";

interface TestUser {
  email: string;
  name: string;
  role_id: string;
  teamRole: string;
  is_owner: boolean;
}

const ALPHA_TEAM = {
  name: "Alpha Team",
  display_name: "Alpha Team",
  description: "Test team A for permission isolation tests",
  status: "active",
  type: "other",
  is_verified: true,
  role_id: "user:*",
};

const ALPHA_USERS: TestUser[] = [
  {
    email: "alpha-owner@test.local",
    name: "Alpha Owner",
    role_id: "user:*",
    teamRole: "team:owner",
    is_owner: true,
  },
  {
    email: "alpha-admin@test.local",
    name: "Alpha Admin",
    role_id: "user:*",
    teamRole: "team:admin",
    is_owner: false,
  },
  {
    email: "alpha-member@test.local",
    name: "Alpha Member",
    role_id: "user:*",
    teamRole: "team:member",
    is_owner: false,
  },
];

const BETA_OPENAI_TEAM = {
  name: "BetaOpenAI Team",
  display_name: "BetaOpenAI Team",
  description: "E2E test team — OpenAI protocol (deepseek via OpenAI-compatible API)",
  status: "active",
  type: "other",
  is_verified: true,
  role_id: "user:*",
};

const BETA_OPENAI_USERS: TestUser[] = [
  {
    email: "beta-openai-owner@test.local",
    name: "BetaOpenAI Owner",
    role_id: "user:*",
    teamRole: "team:owner",
    is_owner: true,
  },
  {
    email: "beta-openai-admin@test.local",
    name: "BetaOpenAI Admin",
    role_id: "user:*",
    teamRole: "team:admin",
    is_owner: false,
  },
  {
    email: "beta-openai-member@test.local",
    name: "BetaOpenAI Member",
    role_id: "user:*",
    teamRole: "team:member",
    is_owner: false,
  },
];

const BETA_ANTHROPIC_TEAM = {
  name: "BetaAnthropic Team",
  display_name: "BetaAnthropic Team",
  description: "E2E test team — Anthropic protocol (deepseek via Anthropic-compatible API)",
  status: "active",
  type: "other",
  is_verified: true,
  role_id: "user:*",
};

const BETA_ANTHROPIC_USERS: TestUser[] = [
  {
    email: "beta-anthropic-owner@test.local",
    name: "BetaAnthropic Owner",
    role_id: "user:*",
    teamRole: "team:owner",
    is_owner: true,
  },
  {
    email: "beta-anthropic-admin@test.local",
    name: "BetaAnthropic Admin",
    role_id: "user:*",
    teamRole: "team:admin",
    is_owner: false,
  },
  {
    email: "beta-anthropic-member@test.local",
    name: "BetaAnthropic Member",
    role_id: "user:*",
    teamRole: "team:member",
    is_owner: false,
  },
];

const BETA_HAIKU_TEAM = {
  name: "BetaHaiku Team",
  display_name: "BetaHaiku Team",
  description:
    "E2E test team — native Anthropic Haiku (for Attachments/vision)",
  status: "active",
  type: "other",
  is_verified: true,
  role_id: "user:*",
};

const BETA_HAIKU_USERS: TestUser[] = [
  {
    email: "beta-haiku-owner@test.local",
    name: "BetaHaiku Owner",
    role_id: "user:*",
    teamRole: "team:owner",
    is_owner: true,
  },
  {
    email: "beta-haiku-admin@test.local",
    name: "BetaHaiku Admin",
    role_id: "user:*",
    teamRole: "team:admin",
    is_owner: false,
  },
  {
    email: "beta-haiku-member@test.local",
    name: "BetaHaiku Member",
    role_id: "user:*",
    teamRole: "team:member",
    is_owner: false,
  },
];

const BETA_GPT4O_TEAM = {
  name: "BetaGPT4o Team",
  display_name: "BetaGPT4o Team",
  description: "E2E test team — native OpenAI GPT-4o (for Attachments/vision)",
  status: "active",
  type: "other",
  is_verified: true,
  role_id: "user:*",
};

const BETA_GPT4O_USERS: TestUser[] = [
  {
    email: "beta-gpt4o-owner@test.local",
    name: "BetaGPT4o Owner",
    role_id: "user:*",
    teamRole: "team:owner",
    is_owner: true,
  },
  {
    email: "beta-gpt4o-admin@test.local",
    name: "BetaGPT4o Admin",
    role_id: "user:*",
    teamRole: "team:admin",
    is_owner: false,
  },
  {
    email: "beta-gpt4o-member@test.local",
    name: "BetaGPT4o Member",
    role_id: "user:*",
    teamRole: "team:member",
    is_owner: false,
  },
];

function createTeamWithUsers(
  teamDef: Record<string, any>,
  users: TestUser[],
  ownerUser: TestUser
): { teamId: string; userIds: Record<string, string> } {
  const teamId = generateId();
  const ownerId = generateId();

  Process("models.__yao.user.Save", {
    email: ownerUser.email,
    name: ownerUser.name,
    status: "active",
    role_id: ownerUser.role_id,
    type: "free",
    locale: "en-us",
    user_id: ownerId,
    password_hash: TEST_PASSWORD,
  });

  Process("models.__yao.team.Save", {
    ...teamDef,
    team_id: teamId,
    owner_id: ownerId,
    contact_email: ownerUser.email,
  });

  Process("models.__yao.member.Save", {
    member_id: generateId(),
    team_id: teamId,
    user_id: ownerId,
    member_type: "user",
    display_name: ownerUser.name,
    email: ownerUser.email,
    role_id: ownerUser.teamRole,
    is_owner: true,
    status: "active",
    joined_at: new Date().toISOString(),
  });

  const userIds: Record<string, string> = {};
  userIds[ownerUser.email] = ownerId;

  for (const u of users) {
    if (u.email === ownerUser.email) continue;
    const uid = generateId();
    Process("models.__yao.user.Save", {
      email: u.email,
      name: u.name,
      status: "active",
      role_id: u.role_id,
      type: "free",
      locale: "en-us",
      user_id: uid,
      password_hash: TEST_PASSWORD,
    });
    Process("models.__yao.member.Save", {
      member_id: generateId(),
      team_id: teamId,
      user_id: uid,
      member_type: "user",
      display_name: u.name,
      email: u.email,
      role_id: u.teamRole,
      is_owner: u.is_owner,
      status: "active",
      joined_at: new Date().toISOString(),
    });
    userIds[u.email] = uid;
  }

  return { teamId, userIds };
}

/**
 * SetupTestUsers - Create test user matrix for unit tests
 * yao run scripts.setup.SetupTestUsers
 */
function SetupTestUsers() {
  log.Info("=== Setting up test users ===");

  const existing = Process("models.__yao.user.Get", {
    wheres: [{ column: "email", value: "alpha-owner@test.local" }],
    limit: 1,
  });
  if (existing && existing.length > 0) {
    log.Info("Test users already exist, skipping.");
    return;
  }

  const alphaOwner = ALPHA_USERS.find((u) => u.is_owner)!;
  const alpha = createTeamWithUsers(ALPHA_TEAM, ALPHA_USERS, alphaOwner);
  log.Info(`Alpha Team created: team_id=${alpha.teamId}`);

  const betaOpenAIOwner = BETA_OPENAI_USERS.find((u) => u.is_owner)!;
  const betaOpenAI = createTeamWithUsers(BETA_OPENAI_TEAM, BETA_OPENAI_USERS, betaOpenAIOwner);
  log.Info(`BetaOpenAI Team created: team_id=${betaOpenAI.teamId}`);

  const betaAnthropicOwner = BETA_ANTHROPIC_USERS.find((u) => u.is_owner)!;
  const betaAnthropic = createTeamWithUsers(BETA_ANTHROPIC_TEAM, BETA_ANTHROPIC_USERS, betaAnthropicOwner);
  log.Info(`BetaAnthropic Team created: team_id=${betaAnthropic.teamId}`);

  const betaHaikuOwner = BETA_HAIKU_USERS.find((u) => u.is_owner)!;
  const betaHaiku = createTeamWithUsers(BETA_HAIKU_TEAM, BETA_HAIKU_USERS, betaHaikuOwner);
  log.Info(`BetaHaiku Team created: team_id=${betaHaiku.teamId}`);

  const betaGPT4oOwner = BETA_GPT4O_USERS.find((u) => u.is_owner)!;
  const betaGPT4o = createTeamWithUsers(BETA_GPT4O_TEAM, BETA_GPT4O_USERS, betaGPT4oOwner);
  log.Info(`BetaGPT4o Team created: team_id=${betaGPT4o.teamId}`);

  // Alpha Team: all roles use mock connector (for Unit/Sandbox tests).
  setupTeamLLMRoles(alpha.teamId, {
    default: { provider: "openai.mock", model: "" },
    vision: { provider: "openai.mock", model: "" },
    heavy: { provider: "openai.mock", model: "" },
    light: { provider: "openai.mock", model: "" },
  });
  log.Info(`Alpha Team LLM roles configured (mock)`);

  // BetaOpenAI Team: OpenAI-compatible protocol (deepseek.v4-flash).
  setupTeamLLMRoles(betaOpenAI.teamId, {
    default: { provider: "deepseek.v4-flash", model: "" },
    vision: { provider: "openai.gpt-4o-mini", model: "" },
    heavy: { provider: "deepseek.v4-flash", model: "" },
    light: { provider: "deepseek.v4-flash", model: "" },
  });
  log.Info(`BetaOpenAI Team LLM roles configured (deepseek-flash openai + gpt-4o-mini)`);

  // BetaAnthropic Team: Anthropic-compatible protocol (deepseek.v4-flash-anthropic).
  setupTeamLLMRoles(betaAnthropic.teamId, {
    default: { provider: "deepseek.v4-flash-anthropic", model: "" },
    vision: { provider: "openai.gpt-4o-mini", model: "" },
    heavy: { provider: "deepseek.v4-flash-anthropic", model: "" },
    light: { provider: "deepseek.v4-flash-anthropic", model: "" },
  });
  log.Info(`BetaAnthropic Team LLM roles configured (deepseek-flash anthropic + gpt-4o-mini)`);

  // BetaHaiku Team: native Anthropic Haiku (for Attachments/vision tests).
  setupTeamLLMRoles(betaHaiku.teamId, {
    default: { provider: "anthropic.haiku", model: "" },
    vision: { provider: "anthropic.haiku", model: "" },
    heavy: { provider: "anthropic.haiku", model: "" },
    light: { provider: "anthropic.haiku", model: "" },
  });
  log.Info(`BetaHaiku Team LLM roles configured (anthropic.haiku)`);

  // BetaGPT4o Team: native OpenAI GPT-4o (for Attachments/vision tests).
  setupTeamLLMRoles(betaGPT4o.teamId, {
    default: { provider: "openai.gpt-4o", model: "" },
    vision: { provider: "openai.gpt-4o", model: "" },
    heavy: { provider: "openai.gpt-4o", model: "" },
    light: { provider: "openai.gpt-4o", model: "" },
  });
  log.Info(`BetaGPT4o Team LLM roles configured (openai.gpt-4o)`);

  log.Info("=== Test users setup completed ===");
}

function setupTeamLLMRoles(
  teamId: string,
  roles: Record<string, { provider: string; model: string }>
) {
  Process(
    "setting.set",
    { scope: "team", team_id: teamId },
    "llm.roles",
    roles
  );
}
