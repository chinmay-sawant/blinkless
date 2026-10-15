"""Loader lock and ownership tests that do not need a shared library."""

import ctypes
from concurrent.futures import ThreadPoolExecutor
from contextlib import ExitStack
from pathlib import Path
import unittest
from unittest import mock

from blinkless import _lib


class LoaderLockTest(unittest.TestCase):
    def test_concurrent_first_use_loads_once_and_locks_abi_calls(self):
        def abi():
            self.assertTrue(_lib._CALL_LOCK.locked())
            return _lib.ABI_VERSION

        library = mock.Mock()
        library.blinkless_abi_version.side_effect = abi
        with ExitStack() as stack:
            stack.enter_context(mock.patch.object(_lib, "_LOADED_LIBRARY", None))
            stack.enter_context(
                mock.patch.object(_lib, "find_library_path", return_value=Path("fake.so"))
            )
            stack.enter_context(mock.patch.object(_lib, "_bind_prototypes"))
            load = stack.enter_context(
                mock.patch.object(_lib.ctypes, "CDLL", return_value=library)
            )
            pool = stack.enter_context(ThreadPoolExecutor(max_workers=8))
            versions = list(pool.map(lambda _: _lib.abi_version(), range(32)))

        self.assertEqual(versions, [_lib.ABI_VERSION] * 32)
        load.assert_called_once_with("fake.so")
        self.assertEqual(library.blinkless_abi_version.call_count, 33)

    def test_version_is_copied_and_freed_under_call_lock(self):
        allocation = ctypes.create_string_buffer(b"test-version")

        def version():
            self.assertTrue(_lib._CALL_LOCK.locked())
            return ctypes.addressof(allocation)

        def release(pointer):
            self.assertTrue(_lib._CALL_LOCK.locked())
            self.assertEqual(
                ctypes.cast(pointer, ctypes.c_void_p).value,
                ctypes.addressof(allocation),
            )
            allocation.value = b"released"

        library = mock.Mock()
        library.blinkless_version.side_effect = version
        library.blinkless_free_string.side_effect = release
        with mock.patch.object(_lib, "load_library", return_value=library):
            result = _lib.library_version_string()

        self.assertEqual(result, "test-version")
        self.assertEqual(allocation.value, b"released")
        library.blinkless_free_string.assert_called_once()
