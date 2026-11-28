"""Isolated installer regression tests; never installs on the test host."""
import os
from pathlib import Path
import pty
import select
import shlex
import socket
import subprocess
import tempfile
import time
import unittest

SCRIPT = Path(__file__).with_name('install.sh').resolve()

class InstallerTests(unittest.TestCase):
    def run_shell(self, body, **env):
        return subprocess.run(['bash', '-c', f'source {shlex.quote(str(SCRIPT))}\n{body}'],
                              env={**os.environ, **env}, text=True, capture_output=True)

    def test_random_port_and_occupied_override(self):
        with socket.socket() as listener:
            listener.bind(('0.0.0.0', 0))
            listener.listen()
            port = listener.getsockname()[1]
            result = self.run_shell(f'free_port {port}')
            self.assertNotEqual(result.returncode, 0)
            result = self.run_shell('free_port 0')
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertNotEqual(int(result.stdout), port)
            self.assertTrue(20000 <= int(result.stdout) <= 60000)

    def test_noninteractive_requires_password_before_installing(self):
        result = self.run_shell('INTERACTIVE=false; ADMIN_PASS=""; choose_credentials')
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('ADMIN_PASS', result.stderr)

    def test_config_quotes_and_launcher_roundtrip(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            dest = root / 'install space $literal%'; dest.mkdir()
            config = root / "config's space" / 'config.json'
            password = 'a b\'"$HOME;`not-a-command`\\100%'
            # A fake panel records arguments; no server, network or service manager.
            fake = dest / 'sing-box-easy'
            fake.write_text('#!/usr/bin/env bash\nprintf "%s\\0" "$@" > args\n')
            fake.chmod(0o755)
            body = '''SUDO=''; KEEP_CONFIG=false; PORT=34567
APP_YML="$INSTALL_DIR/app.yml"; DB_PATH="${SINGBOX_CONFIG%/*}/state.db"
write_config; write_launcher
"$INSTALL_DIR/start-panel"
'''
            result = self.run_shell(body, TMP_DIR=directory, INSTALL_DIR=str(dest),
                                    SINGBOX_CONFIG=str(config), ADMIN_USER='admin $literal', ADMIN_PASS=password)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertIn('auth: "enabled"', (dest / 'app.yml').read_text())
            self.assertIn(str(config), (dest / 'app.yml').read_text())
            args = (dest / 'args').read_bytes().split(b'\0')
            self.assertIn(('-admin_pass_file=' + str(dest / '.bootstrap-password')).encode(), args)
            self.assertNotIn(password.encode(), b' '.join(args))
            self.assertEqual((dest / '.bootstrap-password').read_text(), password)
            self.assertEqual((dest / '.bootstrap-password').stat().st_mode & 0o777, 0o600)
            self.assertNotIn(password, (dest / 'start-panel').read_text())
            self.assertFalse(config.exists())

    def test_upgrade_preserves_existing_configuration(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / 'app.yml'; path.write_text('server:\n  port: "19876"\n# custom settings\n')
            result = self.run_shell('''INTERACTIVE=false; SUDO=''; CORE_VERSION=skip
configure; write_config; test "$ROLE" = keep
''', TMP_DIR=directory, INSTALL_DIR=directory, SINGBOX_VERSION='skip', DEPLOYMENT_ROLE='exit')
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual(path.read_text(), 'server:\n  port: "19876"\n# custom settings\n')
            self.assertIn(':19876/dashboard/settings', result.stdout)

    def test_interactive_progress_password_retry_and_skip(self):
        with tempfile.TemporaryDirectory() as directory:
            destination = str(Path(directory) / 'panel')
            config = str(Path(directory) / 'config.json')
            child, terminal = pty.fork()
            if child == 0:
                body = f'''source {shlex.quote(str(SCRIPT))}
INTERACTIVE=true; SUDO=''; TMP_DIR={shlex.quote(directory)}
exec 3<>/dev/tty
fetch() {{ printf '[{{"tag_name":"v1.14.1"}},{{"tag_name":"v1.15.0-alpha.1"}}]'; }}
configure
'''
                os.execve('/bin/bash', ['bash', '-c', body], {**os.environ, 'INSTALL_DIR': destination,
                          'SINGBOX_CONFIG': config, 'ADMIN_PASS': '', 'SINGBOX_VERSION': ''})
            output = b''
            try:
                def reply(marker, value):
                    nonlocal output
                    end = time.time() + 15
                    pending = b''
                    while marker.encode() not in pending:
                        self.assertLess(time.time(), end, output.decode(errors='replace'))
                        if select.select([terminal], [], [], .2)[0]:
                            chunk = os.read(terminal, 65536); output += chunk; pending += chunk
                    os.write(terminal, value.encode() + b'\n')
                reply('面板安装目录', '')
                reply('面板监听端口', 'invalid')
                reply('面板监听端口', '')
                reply('sing-box 配置文件路径', '')
                reply('管理员用户名', 'operator')
                reply('管理员密码（输入不显示）', 'secret-one')
                reply('再次输入密码', 'secret-two')
                reply('管理员密码（输入不显示）', 'actual-secret')
                reply('再次输入密码', 'actual-secret')
                reply('设备用途', '3')
                reply('选择 sing-box 版本', '0')
                reply('开始安装？', 'y')
                while True:
                    try:
                        chunk = os.read(terminal, 65536)
                        if not chunk: break
                        output += chunk
                    except OSError: break
                _, status = os.waitpid(child, 0)
                self.assertEqual(os.waitstatus_to_exitcode(status), 0, output.decode(errors='replace'))
                text = output.decode(errors='replace')
                self.assertIn('v1.14.1 · 稳定版', text)
                self.assertIn('预发布版', text)
                self.assertIn('/dashboard/settings', text)
                self.assertNotIn('actual-secret', text)
                self.assertNotIn('secret-one', text)
            finally:
                os.close(terminal)

if __name__ == '__main__':
    unittest.main()
