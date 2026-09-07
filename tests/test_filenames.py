from datetime import datetime

from printtoqrview.filenames import generate_filename


def test_generates_name_with_timestamp_and_title():
    ts = datetime(2026, 9, 7, 14, 30, 5)
    name = generate_filename("Rechnung September", timestamp=ts)
    assert name == "20260907-143005_Rechnung_September.pdf"


def test_sanitizes_unsafe_characters():
    ts = datetime(2026, 9, 7, 14, 30, 5)
    name = generate_filename("Bericht: Q3/Ä Ö Ü??.docx", timestamp=ts)
    assert name.startswith("20260907-143005_")
    assert name.endswith(".pdf")
    assert all(c not in name.replace(".pdf", "") for c in "äöü/:? ")


def test_empty_title_falls_back_to_default():
    ts = datetime(2026, 9, 7, 14, 30, 5)
    name = generate_filename("   ", timestamp=ts)
    assert name == "20260907-143005_print.pdf"


def test_two_calls_at_different_times_are_unique():
    name1 = generate_filename("Test", timestamp=datetime(2026, 1, 1, 0, 0, 0))
    name2 = generate_filename("Test", timestamp=datetime(2026, 1, 1, 0, 0, 1))
    assert name1 != name2
