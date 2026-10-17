#!/usr/bin/env python3
import importlib.util
import json
import os
import subprocess
import sys
import tempfile
import unittest

REPO = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
HERDR_TAB_SYNC = os.path.join(REPO, "phone-page", "herdr-tab-sync")
MODULE_PATH = os.path.join(REPO, "phone-page", "tabtasks.py")


def load_module():
    spec = importlib.util.spec_from_file_location("tabtasks", MODULE_PATH)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def write(path, content):
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, "w", encoding="utf-8") as handle:
        handle.write(content)


def tab(tab_id, cwd, workspace="ws", label="tab"):
    return {"tab_id": tab_id, "workspace_id": tab_id.split(":")[0], "workspace_label": workspace, "tab_label": label, "cwd": cwd}


def page(page_id, tab_id, submitted_at=None, title="Page"):
    return {"id": page_id, "title": title, "created_at": "2026-10-10 01:02:03", "herdr_tab_id": tab_id, "submitted_at": submitted_at, "url": f"https://store.example/p/{page_id}"}



class PureFunctionTests(unittest.TestCase):
    def setUp(self):
        self.module = load_module()

    def test_task_for_tab(self):
        pending = {"uuid": "p", "herdrtab": "w1:t1", "status": "pending"}
        waiting = {"uuid": "w", "herdrtab": "w1:t1", "status": "waiting"}
        completed = {"uuid": "c", "herdrtab": "w1:t1", "status": "completed"}
        deleted = {"uuid": "d", "herdrtab": "w1:t1", "status": "deleted"}
        cases = [
            ("pending wins over completed", [completed, pending], "w1:t1", True, "p"),
            ("waiting counts as pending", [waiting], "w1:t1", True, "w"),
            ("completed ignored when pending only", [completed], "w1:t1", True, None),
            ("completed used for attach", [completed], "w1:t1", False, "c"),
            ("deleted never matches", [deleted], "w1:t1", False, None),
            ("other tab never matches", [pending], "w1:t2", False, None),
            ("no herdrtab never matches", [{"uuid": "x", "status": "pending"}], "w1:t1", False, None),
        ]
        for name, tasks, tab_id, pending_only, expected in cases:
            with self.subTest(name):
                task = self.module.task_for_tab(tasks, tab_id, pending_only)
                self.assertEqual(task["uuid"] if task else None, expected)

    def test_project_for(self):
        home = os.path.expanduser("~")
        cases = [
            ("repo folder", "/work/repos/alpha", "alpha"),
            ("trailing slash", "/work/repos/alpha/", "alpha"),
            ("deleted folder", "/work/repos/beta (deleted)", "beta"),
            ("home folder", home, "misc"),
            ("home with slash", home + "/", "misc"),
            ("empty", "", "misc"),
        ]
        for name, cwd, expected in cases:
            with self.subTest(name):
                self.assertEqual(self.module.project_for(cwd), expected)

    def test_delivery_log(self):
        cases = [
            ("empty log", "", (set(), set())),
            ("success", "delivered page=ok tab=w1:t1 at=2026-10-16T01:02:03.000Z\n", ({("ok", "w1:t1")}, set())),
            ("failure", 'page store listener: could not prompt page "bad" tab "w1:t1"\n', (set(), {"bad"})),
            ("ineligible", 'page store listener: no eligible agent for page "bad" tab "w1:t1"\n', (set(), {"bad"})),
            ("unrelated mention", 'other application: page "ok"\n', (set(), set())),
            ("partial success line", "delivered page=ok tab=w1:t1\n", (set(), set())),
            ("page prefix", "delivered page=okay tab=w1:t1 at=2026-10-16T01:02:03.000Z\n", ({("okay", "w1:t1")}, set())),
        ]
        for name, text, expected in cases:
            with self.subTest(name):
                self.assertEqual(self.module.parse_delivery_log(text), expected)

    def test_page_status(self):
        journal = ({("ok", "w1:t1")}, {"bad", "ok"})
        tabs = {"w1:t1"}
        cases = [
            ("success overrides earlier failure", page("ok", "w1:t1", "2026-10-11 00:00:00"), journal, tabs, "answered"),
            ("failure only", page("bad", "w1:t1", "2026-10-11 00:00:00"), journal, tabs, "undelivered"),
            ("no evidence", page("missing", "w1:t1", "2026-10-11 00:00:00"), journal, tabs, "unknown"),
            ("different asking tab", page("ok", "w1:t2", "2026-10-11 00:00:00"), ({("ok", "w1:t1")}, set()), tabs, "unknown"),
            ("open despite log", page("ok", "w1:t1"), journal, tabs, "open"),
            ("tab gone", page("g", "w9:t9"), journal, tabs, "tabgone"),
            ("no tab", page("n", None, "2026-10-11 00:00:00"), journal, tabs, "notab"),
            ("journal unknown", page("ok", "w1:t1", "2026-10-11 00:00:00"), None, tabs, "unknown"),
            ("tabs unknown", page("g", "w9:t9"), journal, None, "unknown"),
        ]
        for name, entry, evidence, tab_ids, expected in cases:
            with self.subTest(name):
                self.assertEqual(self.module.page_status(entry, evidence, tab_ids), expected)
                self.assertEqual(self.module.delivered(entry, evidence) if evidence is not None else False, expected == "answered")

    def test_dry_run_annotation(self):
        cases = [
            ("new annotation", {}, True),
            ("existing annotation", {"annotations": [{"description": "phone-page p answered now"}]}, False),
        ]
        for name, task, expected in cases:
            with self.subTest(name):
                self.assertEqual(self.module.ensure_annotation(task, "phone-page p answered now", True), expected)
                self.assertEqual(task["annotations"], [{"description": "phone-page p answered now"}])

    def test_errors_exit_nonzero(self):
        cases = [
            ("missing executable", ["/nonexistent/tabtasks-test-command"], "cannot run"),
            ("failed process", [sys.executable, "-c", "import sys; sys.stderr.write('failed'); sys.exit(2)"], "exited 2: failed"),
        ]
        for name, argv, message in cases:
            with self.subTest(name):
                with self.assertRaisesRegex(self.module.CommandError, message):
                    self.module.run(argv)


class SyncTests(unittest.TestCase):
    def setUp(self):
        self.module = load_module()
        self.directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.directory.cleanup)
        self.platform = self.directory.name
        write(os.path.join(self.platform, "tab-areas.v2.tsv"), "/work/repos/\twork\tWork Eng\n/work/repos/deep/\tpersonal\tMe\n/finance/\tfinance\tOld finance\n/shared/\tshared\tOld shared\n")
        write(os.path.join(self.platform, "tab-overrides.tsv"), "tab\tarea\towner\tnote\nw2:t1\tsidegig\tSide\tmanual\n")
        write(os.path.join(self.platform, "tab-overrides.v2.tsv"), "gamma\tspecial\tfinance\tFin\n")
        self.rules = self.module.Rules(self.platform)

    def test_classification_and_task_fields(self):
        cases = [
            ("standard area uses route", "w1:t1", "/work/repos/alpha", "ws", "tab", "work", "", False),
            ("per-tab override beats prefix", "w2:t1", "/work/repos/beta (deleted)", "ws", "tab", "sidegig", "Side", False),
            ("cos fallback", "w3:t1", "/other/cos-lab", "ws", "tab", "personal", "Chief of Staff", True),
            ("chief-of-staff beats work prefix", "w2H:tV", "/work/repos/garage/chief-of-staff", "ws", "cos", "personal", "Chief of Staff", True),
            ("shared fallback", "w3:t2", "/shared/project", "ws", "tab", "personal", "Chief of Staff", True),
            ("unmatched fallback", "w3:t3", "/nowhere", "ws", "tab", "personal", "Chief of Staff", True),
            ("finance folder", "w3:t4", "/finance/budget", "ws", "tab", "personal", "Finance", False),
            ("finance label", "w3:t5", "/nowhere", "gamma", "special", "personal", "Finance", False),
            ("longest prefix", "w4:t1", "/work/repos/deep/delta", "ws", "tab", "personal", "", False),
            ("no cwd", "w4:t2", "", "ws", "tab", "personal", "Chief of Staff", True),
        ]
        for name, tab_id, cwd, workspace, label, area, owner, triage in cases:
            with self.subTest(name):
                entry = self.module.Tab(tab_id, tab_id.split(":")[0], workspace, label, cwd)
                self.assertEqual(self.rules.classify(entry), (area, owner))
                self.assertEqual(self.rules.triage(entry), triage)
                fields = self.module.tab_task_fields(entry, area, owner, triage)
                self.assertIn(fields["area"], ("work", "sidegig", "personal"))
                self.assertEqual(fields.get("tags", []), ["triage"] if triage else [])
                self.assertEqual(fields["herdrtab"], tab_id)
                self.assertEqual(fields["sourceid"], tab_id)
                self.assertEqual(fields["source"], "herdr")
                self.assertEqual(fields["status"], "pending")

    def test_per_tab_override_beats_label_and_fallback(self):
        write(os.path.join(self.platform, "tab-overrides.tsv"), "tab\tarea\towner\tnote\nw2:t1\tsidegig\tSide\tmanual\nwJ6:t1\tpersonal\tPlatform\tbackup workspace tab, project misc\n")
        rules = self.module.Rules(self.platform)
        entry = self.module.Tab("wJ6:t1", "wJ6", "gamma", "special", os.path.expanduser("~"))
        self.assertEqual(rules.classify(entry), ("personal", "Platform"))
        self.assertFalse(rules.triage(entry))
        self.assertEqual(self.module.tab_task_fields(entry, *rules.classify(entry))["project"], "misc")

    def test_live_listing_keeps_every_tab_and_uses_current_cwd(self):
        cases = [
            ("tab cwd beats agent cwd", {"cwd": "/work/current"}, [{"tab_id": "w1:t1", "cwd": "/work/old"}], "/work/current"),
            ("agent cwd", {}, [{"tab_id": "w1:t1", "cwd": "/work/agent"}], "/work/agent"),
            ("empty agent cwd does not hide usable cwd", {}, [{"tab_id": "w1:t1", "cwd": ""}, {"tab_id": "w1:t1", "cwd": "/work/agent"}], "/work/agent"),
            ("shell tab without agent", {}, [], ""),
        ]
        for name, extra, agents, expected_cwd in cases:
            with self.subTest(name):
                tabs = self.module.tabs_from_listing(
                    [{"workspace_id": "w1", "label": "workspace"}],
                    [{"tab_id": "w1:t1", "workspace_id": "w1", "label": "tab", **extra}],
                    agents,
                )
                self.assertEqual(list(tabs), ["w1:t1"])
                entry = tabs["w1:t1"]
                self.assertEqual((entry.workspace_id, entry.workspace_label, entry.tab_label, entry.cwd), ("w1", "workspace", "tab", expected_cwd))

    def test_explicit_platform_input_uses_newest_file_and_rejects_missing_data(self):
        root = os.path.join(self.platform, "tabs", "2026-10-16")
        for name, generated, label in [("old.json", "2026-10-16T07:00:00+09:00", "old"), ("new.json", "2026-10-16T09:00:00+09:00", "new")]:
            write(os.path.join(root, name), json.dumps({"generated": generated, "tabs": [tab("w1:t1", "/work/repos/alpha", "alpha", label)]}))
        tabs, source = self.module.load_tabs(self.platform, live=False)
        self.assertEqual(tabs["w1:t1"].tab_label, "new")
        self.assertIn("2 files, generated", source)
        cases = [
            ("missing tab id", [{"cwd": "/work"}]),
            ("missing cwd", [{"tab_id": "w1:t1"}]),
            ("empty", []),
        ]
        os.remove(os.path.join(root, "old.json"))
        for name, entries in cases:
            with self.subTest(name):
                write(os.path.join(root, "new.json"), json.dumps({"tabs": entries}))
                with self.assertRaises(self.module.CommandError):
                    self.module.load_tabs(self.platform, live=False)

    def test_input_mode_help(self):
        completed = subprocess.run([HERDR_TAB_SYNC, "--help"], capture_output=True, text=True)
        self.assertEqual(completed.returncode, 0, completed.stderr)
        self.assertIn("--live | --platform", completed.stdout)
        self.assertIn("read live Herdr tabs (default)", completed.stdout)


class PlanningTests(unittest.TestCase):
    def setUp(self):
        self.module = load_module()
        self.directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.directory.cleanup)
        self.tabs = {
            tab_id: self.module.Tab(**tab(tab_id, "/work/repos/alpha", "alpha", label))
            for tab_id, label in [("w1:t2", "new"), ("w1:t1", "kept")]
        }
        self.tasks = [
            {"uuid": "keep", "herdrtab": "w1:t1", "status": "waiting"},
            {"uuid": "old", "herdrtab": "w1:t2", "status": "completed"},
            {"uuid": "gone", "herdrtab": "w9:t9", "status": "pending"},
            {"uuid": "done", "herdrtab": "w8:t8", "status": "completed"},
            {"uuid": "deleted", "herdrtab": "w7:t7", "status": "deleted"},
            {"uuid": "plain", "status": "pending"},
        ]

    def test_plan_from_platform_files(self):
        root = os.path.join(self.directory.name, "tabs", "2026-10-16", "work.json")
        write(root, json.dumps({"tabs": [tab("w1:t1", "/work/repos/alpha"), tab("w1:t2", "/work/repos/beta")]}))
        tabs, source = self.module.load_tabs(self.directory.name, live=False)
        before = json.dumps(self.tasks)
        cases = [
            ("platform files", tabs),
            ("live listing data", self.tabs),
        ]
        self.assertTrue(source.startswith("platform files "))
        for name, entries in cases:
            with self.subTest(name):
                current, closed = self.module.sync_plan(entries, self.tasks)
                self.assertEqual([(action, entry.tab_id) for action, entry in current], [("keep", "w1:t1"), ("create", "w1:t2")])
                self.assertEqual([task["uuid"] for task in closed], ["gone"])
                self.assertEqual(json.dumps(self.tasks), before)

    def test_sync_plan_is_idempotent(self):
        for attempt in ("first", "second"):
            with self.subTest(attempt):
                current, closed = self.module.sync_plan(self.tabs, self.tasks)
                self.assertEqual([(action, entry.tab_id) for action, entry in current], [("keep", "w1:t1"), ("create" if attempt == "first" else "keep", "w1:t2")])
                self.assertEqual([task["uuid"] for task in closed], ["gone"] if attempt == "first" else [])
                for action, entry in current:
                    if action == "create":
                        self.tasks.append({"uuid": "new", **self.module.tab_task_fields(entry, "personal", "Chief of Staff")})
                for task in closed:
                    task["status"] = "completed"
        self.assertEqual(len(self.tasks), 7)
        self.assertEqual([task["status"] for task in self.tasks if task["uuid"] == "old"], ["completed"])

    def test_attach_annotates_once(self):
        journal = ({("answered", "w1:t1")}, {"failed"})
        cases = [
            ("open", page("open", "w1:t1"), ["phone-page open https://store.example/p/open"]),
            ("delivered answer", page("answered", "w1:t1", "2026-10-11 00:00:00"), ["phone-page answered https://store.example/p/answered", "phone-page answered answered 2026-10-11 00:00:00"]),
            ("failed delivery", page("failed", "w1:t1", "2026-10-11 00:00:00"), ["phone-page failed https://store.example/p/failed"]),
        ]
        for name, entry, expected in cases:
            with self.subTest(name):
                task = {"uuid": "keep"}
                for attempt in ("first", "second"):
                    changed = [self.module.ensure_annotation(task, text, True) for text in self.module.page_annotations(entry, journal)]
                    self.assertEqual(changed, [attempt == "first"] * len(expected))
                    self.assertEqual(self.module.annotations(task), expected)

    def test_listener_log_failure_skips_answered_marking(self):
        entry = page("answered", "w1:t1", "2026-10-11 00:00:00")
        cases = [
            ("unavailable journal", None),
            ("no delivery evidence", (set(), set())),
            ("different asking tab", ({("answered", "w2:t2")}, set())),
        ]
        for name, journal in cases:
            with self.subTest(name):
                self.assertEqual(self.module.page_annotations(entry, journal), ["phone-page answered https://store.example/p/answered"])

    def test_unmatched_and_completed_tasks(self):
        cases = [
            ("pending tab", "w1:t1", "keep"),
            ("completed tab", "w1:t2", "old"),
            ("unknown tab", "w6:t6", None),
            ("deleted task", "w7:t7", None),
        ]
        for name, tab_id, expected in cases:
            with self.subTest(name):
                matched = self.module.task_for_tab(self.tasks, tab_id, pending_only=False)
                self.assertEqual(matched["uuid"] if matched else None, expected)

    def test_statuses_and_actions(self):
        entries = [page("answered", "w1:t1", "2026-10-11 00:00:00"), page("undelivered", "w1:t1", "2026-10-11 00:00:00"), page("open", "w1:t1"), page("tabgone", "w9:t9"), page("notab", None), page("unknown", "w1:t1", "2026-10-11 00:00:00")]
        journal = ({("answered", "w1:t1")}, {"undelivered", "open"})
        expected = [("answered", "answered", "delete"), ("undelivered", "undelivered", "none"), ("open", "open", "none"), ("tabgone", "tabgone", "delete"), ("notab", "notab", "delete"), ("unknown", "unknown", "none")]
        cases = [
            ("known lookups", journal, {"w1:t1"}, expected),
            ("journal unavailable", None, {"w1:t1"}, [(entry["id"], "unknown", "none") for entry in entries]),
            ("tabs unavailable", journal, None, [(entry["id"], "unknown", "none") for entry in entries]),
        ]
        before = json.dumps(entries)
        for name, evidence, tab_ids, expected_plan in cases:
            with self.subTest(name):
                plan = self.module.close_plan(entries, evidence, tab_ids)
                self.assertEqual([(entry["id"], status, action) for entry, status, action in plan], expected_plan)
                self.assertNotIn("open", [entry["id"] for entry, _, action in plan if action == "delete"])
                self.assertTrue(all(action == "none" for _, status, action in plan if status == "unknown"))
                self.assertEqual(json.dumps(entries), before)

    def test_log_appends_across_runs(self):
        path = os.path.join(self.directory.name, "close-log.tsv")
        entries = [page("open", "w1:t1"), page("answered", "w1:t1", "2026-10-11 00:00:00")]
        journal = ({("answered", "w1:t1")}, set())
        for mode in ("dry-run", "live"):
            with self.subTest(mode):
                rows = [("2026-10-16T00:00:00+09:00", entry["id"], "work", status, action, mode) for entry, status, action in self.module.close_plan(entries, journal, {"w1:t1"})]
                self.module.append_tsv(path, self.module.CLOSE_LOG_HEADER, rows)
        self.assertEqual(self.module.read_rows(path), [list(self.module.CLOSE_LOG_HEADER), ["2026-10-16T00:00:00+09:00", "open", "work", "open", "none", "dry-run"], ["2026-10-16T00:00:00+09:00", "answered", "work", "answered", "delete", "dry-run"], ["2026-10-16T00:00:00+09:00", "open", "work", "open", "none", "live"], ["2026-10-16T00:00:00+09:00", "answered", "work", "answered", "delete", "live"]])

    def test_dry_run_unmatched_report_is_separate(self):
        cases = [("unmatched.tsv", [("Live question", "Chief of Staff", "live")]), ("unmatched.dryrun.tsv", [("Dry question", "Finance", "dry")])]
        for name, rows in cases:
            with self.subTest(name):
                self.module.write_tsv(os.path.join(self.directory.name, name), self.module.UNMATCHED_HEADER, rows)
        for name, rows in cases:
            with self.subTest(name):
                self.assertEqual(self.module.read_rows(os.path.join(self.directory.name, name)), [list(self.module.UNMATCHED_HEADER), *[list(row) for row in rows]])


if __name__ == "__main__":
    unittest.main()
