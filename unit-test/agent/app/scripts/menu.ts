/**
 * Menu Management Scripts
 */

import { Exception, Process } from "@yao/runtime";

// Declare global Authorized function (provided by Yao runtime)
declare function Authorized(): AuthorizedInfo | null;

/** Authorized info structure */
interface AuthorizedInfo {
  user_id?: string | number;
  team_id?: string | number;
  scope?: string;
  constraints?: {
    owner_only?: boolean;
    creator_only?: boolean;
    editor_only?: boolean;
    team_only?: boolean;
    extra?: Record<string, any>;
  };
}

/** Menu item structure (database) */
interface MenuRecord {
  id?: number;
  menu_id: string;
  parent?: string | null; // Parent menu_id (string reference)
  name: Record<string, string>; // i18n: { "en": "Dashboard", "zh-cn": "数据看板" }
  description?: Record<string, string> | null;
  icon?: string | { name: string; size?: number } | null;
  path?: string | null;
  type: "items" | "setting" | "quick";
  sort?: number;
  status?: "enabled" | "disabled";
  public?: boolean;
  share?: "private" | "team";
  extra?: Record<string, any> | null;
  // Permission fields
  __yao_created_by?: string | number;
  __yao_team_id?: string | number;
}

/** Menu item structure (output with children) */
interface MenuItem {
  name: string;
  icon?: string | { name: string; size?: number };
  path?: string;
  sort?: number;
  children?: MenuItem[];
}

/** Menu output structure (matches flow output) */
interface MenuOutput {
  items: MenuItem[];
  setting: MenuItem[];
  quick: MenuItem[];
}

/**
 * Get menu data for a specific locale (all enabled menus, no permission filter)
 * @param locale The locale (e.g., "en", "zh-cn")
 * @returns Menu output with items, setting, quick
 * @test yao run scripts.menu.Get en
 * @test yao run scripts.menu.Get zh-cn
 */
function Get(locale: string = "en"): MenuOutput {
  // Get authorized info
  const authInfo = Authorized() as AuthorizedInfo | null;

  // Build auth filters based on login context
  const wheres = buildAuthFilters(authInfo);

  // Get all enabled menus with auth filters
  const allMenus = Process("models.menu.Get", {
    wheres,
    orders: [{ column: "sort", option: "asc" }],
  }) as MenuRecord[];

  return buildMenuTree(allMenus, locale);
}

/**
 * Build auth filters based on authorized info
 * Logic:
 * - If team login (team_id is set): show team records (__yao_team_id = team_id) OR public
 * - If personal login (team_id is null): show personal records (__yao_team_id IS NULL AND __yao_created_by = user_id) OR public
 *
 * Key insight: Same user with team_id=null vs team_id=xxx are TWO different login contexts.
 * Personal records have __yao_team_id = NULL, team records have __yao_team_id = team_id.
 */
function buildAuthFilters(authInfo: AuthorizedInfo | null): any[] {
  const wheres: any[] = [{ column: "status", value: "enabled" }];

  // No auth info - only public menus
  if (!authInfo) {
    wheres.push({ column: "public", value: true });
    return wheres;
  }

  const userId = authInfo.user_id;
  const teamId = authInfo.team_id;

  // Build permission filter based on login context
  const permissionWheres: any[] = [
    { column: "public", value: true }, // Always include public menus
  ];

  if (teamId) {
    // Team login: show team's menus (where __yao_team_id = teamId)
    permissionWheres.push({
      column: "__yao_team_id",
      value: teamId,
      method: "orwhere",
    });
  } else if (userId) {
    // Personal login: show user's personal menus (where __yao_team_id IS NULL AND __yao_created_by = userId)
    permissionWheres.push({
      wheres: [
        { column: "__yao_team_id", op: "null" },
        { column: "__yao_created_by", value: userId },
      ],
      method: "orwhere",
    });
  }

  wheres.push({ wheres: permissionWheres });

  return wheres;
}

/**
 * Build menu tree from flat list using menu_id as parent reference
 * @param menus All menu records
 * @param locale The locale for localization
 * @returns Menu output with tree structure
 */
function buildMenuTree(menus: MenuRecord[], locale: string): MenuOutput {
  // Create a map for quick lookup by menu_id
  const menuMap = new Map<string, MenuRecord & { children?: MenuRecord[] }>();
  menus.forEach((menu) => {
    menuMap.set(menu.menu_id, { ...menu, children: [] });
  });

  // Build tree structure
  const rootMenus: (MenuRecord & { children?: MenuRecord[] })[] = [];

  menus.forEach((menu) => {
    const menuWithChildren = menuMap.get(menu.menu_id);
    if (!menuWithChildren) return;

    if (menu.parent && menuMap.has(menu.parent)) {
      // Has parent, add to parent's children
      const parent = menuMap.get(menu.parent);
      if (parent) {
        parent.children = parent.children || [];
        parent.children.push(menuWithChildren);
      }
    } else {
      // No parent, it's a root menu
      rootMenus.push(menuWithChildren);
    }
  });

  // Group by type and format output
  const output: MenuOutput = {
    items: [],
    setting: [],
    quick: [],
  };

  rootMenus.forEach((menu) => {
    const formatted = formatMenuItem(menu, locale);
    if (menu.type === "items") {
      output.items.push(formatted);
    } else if (menu.type === "setting") {
      output.setting.push(formatted);
    } else if (menu.type === "quick") {
      output.quick.push(formatted);
    }
  });

  return output;
}

/**
 * Format menu record to output item
 */
function formatMenuItem(
  menu: MenuRecord & { children?: MenuRecord[] },
  locale: string
): MenuItem {
  const result: MenuItem = {
    name: localizeName(menu.name, locale),
    sort: menu.sort ?? 0,
  };

  // Add icon if present
  if (menu.icon) {
    result.icon = menu.icon;
  }

  // Add path if present
  if (menu.path) {
    result.path = menu.path;
  }

  // Add children recursively
  if (menu.children && menu.children.length > 0) {
    // Sort children by sort field
    menu.children.sort((a, b) => (a.sort ?? 0) - (b.sort ?? 0));
    result.children = menu.children.map((child) =>
      formatMenuItem(child as MenuRecord & { children?: MenuRecord[] }, locale)
    );
  }

  return result;
}

/**
 * Localize name based on locale with fallback
 * e.g., "zh" will match "zh-cn", "zh-tw", etc.
 */
function localizeName(
  name: string | Record<string, string>,
  locale: string
): string {
  if (typeof name === "string") {
    return name;
  }

  // Exact match
  if (name[locale]) {
    return name[locale];
  }

  // Fallback: try to match by language prefix (e.g., "zh" matches "zh-cn")
  const localeLower = locale.toLowerCase();
  for (const key of Object.keys(name)) {
    const keyLower = key.toLowerCase();
    // Match prefix: "zh" matches "zh-cn", "zh-tw"
    if (
      keyLower.startsWith(localeLower + "-") ||
      localeLower.startsWith(keyLower + "-")
    ) {
      return name[key];
    }
    // Match same prefix: "zh" matches "zh", "zh-cn" matches "zh"
    if (
      keyLower === localeLower ||
      keyLower.split("-")[0] === localeLower.split("-")[0]
    ) {
      return name[key];
    }
  }

  // Final fallback: English or first available
  return name["en"] || Object.values(name)[0] || "";
}

/**
 * Create a new menu item
 * @param data Menu item data
 * @returns Created menu ID
 * @test yao run scripts.menu.Create '{"menu_id":"test","name":{"en":"Test","zh-cn":"测试"},"type":"items"}'
 */
function Create(data: MenuRecord): number {
  // Validate required fields
  if (!data.menu_id) {
    throw new Exception("menu_id is required", 400);
  }
  if (!data.name || Object.keys(data.name).length === 0) {
    throw new Exception("name is required", 400);
  }
  if (!data.type) {
    throw new Exception("type is required", 400);
  }

  // Set defaults
  data.sort = data.sort ?? 0;
  data.status = data.status ?? "enabled";
  data.public = data.public ?? false;
  data.share = data.share ?? "private";

  return Process("models.menu.Create", data);
}

/**
 * Update a menu item
 * @param id Menu ID (primary key)
 * @param data Menu item data
 * @returns Updated menu
 * @test yao run scripts.menu.Update 1 '{"name":{"en":"Updated"}}'
 */
function Update(id: number, data: Partial<MenuRecord>): void {
  if (!id) {
    throw new Exception("id is required", 400);
  }

  // Check permission before update
  checkMenuPermission(id);

  Process("models.menu.Update", id, data);
}

/**
 * Save a menu item (create or update)
 * @param data Menu item data
 * @returns Menu ID
 * @test yao run scripts.menu.Save '{"menu_id":"test","name":{"en":"Test"},"type":"items"}'
 */
function Save(data: MenuRecord): number {
  if (!data.menu_id) {
    throw new Exception("menu_id is required", 400);
  }

  // Check if menu exists by menu_id
  const existing = Process("models.menu.Get", {
    wheres: [{ column: "menu_id", value: data.menu_id }],
    limit: 1,
  }) as MenuRecord[];

  if (existing && existing.length > 0) {
    // Check permission before update
    checkMenuPermission(existing[0].id!);
    data.id = existing[0].id;
  }

  return Process("models.menu.Save", data);
}

/**
 * Delete a menu item
 * @param id Menu ID (primary key)
 * @test yao run scripts.menu.Delete 1
 */
function Delete(id: number): void {
  if (!id) {
    throw new Exception("id is required", 400);
  }

  // Check permission before delete
  checkMenuPermission(id);

  Process("models.menu.Delete", id);
}

/**
 * Check if user has permission to modify a menu
 * @param id Menu ID
 */
function checkMenuPermission(id: number): void {
  const authInfo = Authorized() as AuthorizedInfo | null;

  // No auth info - allow (for admin/system calls)
  if (!authInfo) {
    return;
  }

  const constraints = authInfo.constraints || {};

  // No constraints - allow
  if (!constraints.owner_only && !constraints.team_only) {
    return;
  }

  // Get menu record
  const menu = Process("models.menu.Find", id, {
    select: ["id", "public", "share", "__yao_created_by", "__yao_team_id"],
  }) as MenuRecord;

  if (!menu) {
    throw new Exception("Menu not found", 404);
  }

  // Owner only: check if user owns the menu
  if (constraints.owner_only && authInfo.user_id) {
    if (menu.__yao_created_by === authInfo.user_id) {
      return;
    }
  }

  // Team only: check if menu belongs to user's team
  if (constraints.team_only && authInfo.team_id) {
    if (menu.__yao_team_id === authInfo.team_id) {
      return;
    }
  }

  throw new Exception("No permission to modify this menu", 403);
}

/**
 * Import menu from flow output format
 * @param data Menu data in flow output format
 * @param options Import options
 * @returns Import result
 * @test yao run scripts.menu.Import '{"items":[{"name":"Test","path":"/test"}],"setting":[],"quick":[]}' '{"locale":"en"}'
 */
function Import(
  data: { items?: any[]; setting?: any[]; quick?: any[] },
  options: { locale?: string; clear?: boolean } = {}
): { success: number; failed: number } {
  const locale = options.locale || "en";
  let success = 0;
  let failed = 0;

  // Clear existing menus if requested
  if (options.clear) {
    Process("models.menu.DeleteWhere", {
      wheres: [{ column: "id", op: ">", value: 0 }],
    });
  }

  // Import items
  const importItems = (
    items: any[],
    type: "items" | "setting" | "quick",
    parentMenuId?: string
  ) => {
    items?.forEach((item, index) => {
      try {
        const menuId = item.menu_id || generateMenuId(item.name, type, index);
        const menuData: MenuRecord = {
          menu_id: menuId,
          name:
            typeof item.name === "string" ? { [locale]: item.name } : item.name,
          description: item.description
            ? typeof item.description === "string"
              ? { [locale]: item.description }
              : item.description
            : null,
          icon: item.icon,
          path: item.path,
          type,
          sort: item.sort ?? index,
          status: "enabled",
          public: item.public ?? false,
          share: item.share ?? "private",
          parent: parentMenuId ?? null,
        };

        Save(menuData);
        success++;

        // Import children recursively
        if (item.children && Array.isArray(item.children)) {
          importItems(item.children, type, menuId);
        }
      } catch (e) {
        console.error(`Failed to import menu item: ${item.name}`, e);
        failed++;
      }
    });
  };

  importItems(data.items || [], "items");
  importItems(data.setting || [], "setting");
  importItems(data.quick || [], "quick");

  return { success, failed };
}

/**
 * Export menu to flow output format
 * @param locale The locale for export
 * @returns Menu data in flow output format
 * @test yao run scripts.menu.Export en
 */
function Export(locale: string = "en"): MenuOutput {
  return Get(locale);
}

/**
 * Setup menus from seed file
 * @test yao run scripts.menu.Setup
 */
function Setup(): {
  total: number;
  success: number;
  ignore: number;
  failure: number;
} {
  console.log("Setting up menus from seed file...");

  const options = {
    chunk_size: 100,
    duplicate: "ignore",
    mode: "each",
  };

  const result = Process("seeds.import", "menus.csv", "menu", options);

  console.log(
    `Menus import completed: Total=${result.total}, Success=${result.success}, Ignored=${result.ignore}, Failed=${result.failure}`
  );

  if (result.errors && result.errors.length > 0) {
    console.error("Menus import errors:", result.errors);
  }

  return result;
}

/**
 * Reset menus - Clear system menus and optionally reimport from seed
 * User-created menus are preserved (identified by menu_id pattern)
 * System menu_ids start with: items_, setting_, quick_
 * @param reimport boolean whether to reimport seed data after clearing
 * @test yao run scripts.menu.Reset
 * @test yao run scripts.menu.Reset true
 */
function Reset(reimport: boolean = false): void {
  console.log("Resetting menus...");

  // Get all system menu_ids from seed file to identify what to delete
  const systemMenuIds = [
    "items_chat",
    "items_assistants",
    "items_mission_control",
    "items_api_keys",
    "items_kb",
    "setting_profile",
    // Legacy menu_ids (for cleanup)
    "setting_main",
    "setting_team",
    "setting_api_keys",
    "setting_usage",
  ];

  // Soft delete only system menus by menu_id, preserve user-created menus
  // Use DeleteWhere (soft delete) instead of DestroyWhere (hard delete)
  // DeleteWhere requires QueryParam format: { wheres: [{ column: "...", value: "..." }] }
  let deleted = 0;
  for (const menuId of systemMenuIds) {
    const count = Process("models.menu.DeleteWhere", {
      wheres: [{ column: "menu_id", value: menuId }],
    });
    deleted += count || 0;
  }
  console.log(`Soft deleted ${deleted} system menus (user menus preserved)`);

  // Optionally reimport seed data
  if (reimport) {
    console.log("Reimporting menus from seed data...");
    Setup();
  }

  console.log("Menus reset completed");
}

// ============ Helper Functions ============

/**
 * Generate a unique menu_id from name
 */
function generateMenuId(
  name: string | Record<string, string>,
  type: string,
  index: number
): string {
  const baseName =
    typeof name === "string"
      ? name
      : name["en"] || Object.values(name)[0] || "menu";
  const slug = baseName
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "_")
    .replace(/^_|_$/g, "");
  return `${type}_${slug}_${index}_${Date.now()}`;
}
