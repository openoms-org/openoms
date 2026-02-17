#!/usr/bin/env node
/**
 * OpenOMS Discord Server Setup Script
 *
 * Usage:
 *   1. Create a Discord bot at https://discord.com/developers/applications
 *   2. Enable "Server Members Intent" and "Message Content Intent" in bot settings
 *   3. Invite bot to your server with Administrator permissions:
 *      https://discord.com/oauth2/authorize?client_id=YOUR_APP_ID&permissions=8&scope=bot
 *   4. Run:
 *      DISCORD_BOT_TOKEN=your_token DISCORD_GUILD_ID=your_server_id npm run setup
 *
 * The script is idempotent — running it again won't create duplicates.
 */

import {
  Client,
  GatewayIntentBits,
  ChannelType,
  PermissionFlagsBits,
  Colors,
} from "discord.js";

const TOKEN = process.env.DISCORD_BOT_TOKEN;
const GUILD_ID = process.env.DISCORD_GUILD_ID;

if (!TOKEN || !GUILD_ID) {
  console.error(
    "Missing env vars. Usage:\n  DISCORD_BOT_TOKEN=xxx DISCORD_GUILD_ID=yyy npm run setup"
  );
  process.exit(1);
}

// ── Configuration ────────────────────────────────────────────────

const ROLES = [
  {
    name: "Maintainer",
    color: Colors.Red,
    hoist: true,
    mentionable: true,
    reason: "Core team / project maintainers",
  },
  {
    name: "Contributor",
    color: Colors.Green,
    hoist: true,
    mentionable: true,
    reason: "People who contributed code or docs",
  },
  {
    name: "Uzytkownik",
    color: Colors.Blue,
    hoist: false,
    mentionable: false,
    reason: "Regular community members / users",
  },
];

const CATEGORIES = [
  {
    name: "INFORMACJE",
    channels: [
      {
        name: "ogloszenia",
        topic: "Oficjalne ogłoszenia projektu OpenOMS. Tylko maintainerzy mogą pisać.",
        readOnly: true,
      },
      {
        name: "roadmap",
        topic: "Aktualny roadmap i postępy prac. https://github.com/openoms-org/openoms/blob/main/ROADMAP.md",
        readOnly: true,
      },
      {
        name: "changelog",
        topic: "Logi zmian z każdego release'u. Automatyczne powiadomienia z GitHub Actions.",
        readOnly: true,
      },
    ],
  },
  {
    name: "SPOLECZNOSC",
    channels: [
      {
        name: "general",
        topic: "Ogólne rozmowy o OpenOMS, e-commerce i zarządzaniu zamówieniami.",
      },
      {
        name: "pokaz-swoje",
        topic: "Pochwal się swoją instalacją OpenOMS, customizacjami, integracjami.",
      },
      {
        name: "pomysly",
        topic: "Propozycje nowych funkcji i usprawnień. Głosuj reakcjami!",
      },
    ],
  },
  {
    name: "POMOC",
    channels: [
      {
        name: "instalacja",
        topic: "Pomoc z instalacją: Docker, Go, Node.js, PostgreSQL, konfiguracja .env",
      },
      {
        name: "konfiguracja",
        topic: "Ustawienia tenanta, SMTP, CORS, JWT, szyfrowanie, RBAC, 2FA",
      },
      {
        name: "integracje",
        topic: "Allegro, InPost, DHL, marketplace'y, feedy IOF, API webhooks",
      },
    ],
  },
  {
    name: "ROZWOJ",
    channels: [
      {
        name: "contributing",
        topic: "Dyskusje o kodzie, architekturze, PR review. CONTRIBUTING.md: https://github.com/openoms-org/openoms/blob/main/CONTRIBUTING.md",
      },
      {
        name: "bugs",
        topic: "Zgłaszanie i dyskusja o bugach. Dla oficjalnych zgłoszeń użyj GitHub Issues.",
      },
      {
        name: "pull-requests",
        topic: "Dyskusje o otwartych PR-ach i code review.",
      },
    ],
  },
  {
    name: "BOTY",
    channels: [
      {
        name: "github-feed",
        topic: "Automatyczne powiadomienia: nowe release'y, issues, pull requesty z GitHub.",
        readOnly: true,
      },
    ],
  },
];

const WELCOME_MESSAGE = `# Witaj na serwerze OpenOMS!

Open-source Order Management System dla polskiego e-commerce.

## Przydatne linki
- **GitHub:** https://github.com/openoms-org/openoms
- **Strona:** https://openoms.org
- **Dokumentacja:** https://github.com/openoms-org/openoms/tree/main/docs
- **Zgłoś bug:** https://github.com/openoms-org/openoms/issues/new

## Kanały
- **#general** — ogólne rozmowy
- **#instalacja** — pomoc z setupem
- **#integracje** — Allegro, InPost, marketplace'y
- **#contributing** — chcesz pomóc? Zacznij tutaj!
- **#pomysly** — zaproponuj nową funkcję

## Role
- **@Maintainer** — core team
- **@Contributor** — kontrybutorzy kodu
- **@Uzytkownik** — użytkownicy OpenOMS

Licencja: AGPLv3 (core) + MIT (SDK packages)
`;

// ── Script ───────────────────────────────────────────────────────

const client = new Client({
  intents: [GatewayIntentBits.Guilds, GatewayIntentBits.GuildMembers],
});

async function findOrCreate(collection, name, createFn) {
  const existing = collection.cache.find(
    (item) => item.name.toLowerCase() === name.toLowerCase()
  );
  if (existing) {
    console.log(`  [skip] "${name}" already exists`);
    return existing;
  }
  const created = await createFn();
  console.log(`  [created] "${name}"`);
  return created;
}

async function run() {
  await client.login(TOKEN);
  const guild = await client.guilds.fetch(GUILD_ID);
  console.log(`Connected to: ${guild.name} (${guild.memberCount} members)\n`);

  // ── Roles ──
  console.log("=== ROLES ===");
  await guild.roles.fetch();
  const roleMap = {};

  for (const roleDef of ROLES) {
    const role = await findOrCreate(guild.roles, roleDef.name, () =>
      guild.roles.create({
        name: roleDef.name,
        color: roleDef.color,
        hoist: roleDef.hoist,
        mentionable: roleDef.mentionable,
        reason: roleDef.reason,
      })
    );
    roleMap[roleDef.name] = role;
  }

  const maintainerRole = roleMap["Maintainer"];
  const everyoneRole = guild.roles.everyone;

  // ── Categories & Channels ──
  console.log("\n=== CHANNELS ===");
  await guild.channels.fetch();

  for (const catDef of CATEGORIES) {
    console.log(`\n[category] ${catDef.name}`);

    const category = await findOrCreate(guild.channels, catDef.name, () =>
      guild.channels.create({
        name: catDef.name,
        type: ChannelType.GuildCategory,
      })
    );

    for (const chDef of catDef.channels) {
      const channel = await findOrCreate(guild.channels, chDef.name, () =>
        guild.channels.create({
          name: chDef.name,
          type: ChannelType.GuildText,
          parent: category.id,
          topic: chDef.topic,
        })
      );

      // Move to correct category if it exists but is in wrong category
      if (channel.parentId !== category.id) {
        await channel.setParent(category.id, { lockPermissions: false });
        console.log(`  [moved] "${chDef.name}" -> ${catDef.name}`);
      }

      // Set topic if missing
      if (chDef.topic && channel.topic !== chDef.topic) {
        await channel.setTopic(chDef.topic);
      }

      // Read-only channels: only Maintainers can send
      if (chDef.readOnly) {
        await channel.permissionOverwrites.set([
          {
            id: everyoneRole.id,
            deny: [PermissionFlagsBits.SendMessages],
            allow: [PermissionFlagsBits.ViewChannel],
          },
          {
            id: maintainerRole.id,
            allow: [
              PermissionFlagsBits.SendMessages,
              PermissionFlagsBits.ManageMessages,
            ],
          },
        ]);
        console.log(`  [perms] "${chDef.name}" -> read-only (Maintainer can post)`);
      }
    }
  }

  // ── Welcome message in #ogloszenia ──
  console.log("\n=== WELCOME MESSAGE ===");
  const announceChannel = guild.channels.cache.find(
    (c) => c.name === "ogloszenia"
  );
  if (announceChannel) {
    const messages = await announceChannel.messages.fetch({ limit: 5 });
    const hasWelcome = messages.some((m) =>
      m.content.includes("Witaj na serwerze OpenOMS")
    );
    if (!hasWelcome) {
      await announceChannel.send(WELCOME_MESSAGE);
      console.log("  [sent] Welcome message in #ogloszenia");
    } else {
      console.log("  [skip] Welcome message already exists");
    }
  }

  // ── Server settings ──
  console.log("\n=== SERVER SETTINGS ===");
  const generalChannel = guild.channels.cache.find(
    (c) => c.name === "general"
  );

  // Set system channel to #general
  if (generalChannel && guild.systemChannelId !== generalChannel.id) {
    await guild.setSystemChannel(generalChannel.id);
    console.log("  [set] System channel -> #general");
  }

  // Set server description
  const description =
    "OpenOMS — open-source Order Management System dla polskiego e-commerce. AGPLv3 + MIT.";
  if (guild.description !== description) {
    try {
      await guild.setDescription(description);
      console.log("  [set] Server description");
    } catch {
      console.log(
        "  [skip] Server description (requires Community or boost level)"
      );
    }
  }

  console.log("\n=== DONE ===");
  console.log("Server setup complete!\n");
  console.log("Next steps:");
  console.log(
    "  1. Set up GitHub webhook: repo Settings > Webhooks > Discord webhook URL"
  );
  console.log(
    "     URL format: https://discord.com/api/webhooks/WEBHOOK_ID/WEBHOOK_TOKEN/github"
  );
  console.log("     Target channel: #github-feed");
  console.log(
    "  2. Assign @Maintainer role to core team members manually"
  );
  console.log("  3. Set server icon and banner in Discord settings");

  client.destroy();
}

run().catch((err) => {
  console.error("Setup failed:", err.message);
  client.destroy();
  process.exit(1);
});
