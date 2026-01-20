#!/usr/bin/env python3
"""Generate reference images from test case definitions using Cairo."""

import json
from pathlib import Path

import cairo


def main():
    # paths relative to go-render module root
    input_path = Path("testdata/testcases.json")
    output_dir = Path("testdata/reference")
    output_dir.mkdir(parents=True, exist_ok=True)

    with open(input_path) as f:
        data = json.load(f)

    for tc in data["testcases"]:
        render_testcase(tc, output_dir)
        print(f"  generated {tc['name']}.png")


def render_testcase(tc, output_dir):
    name = tc["name"]
    w, h = tc["width"], tc["height"]

    # create 8-bit alpha surface (grayscale coverage)
    surface = cairo.ImageSurface(cairo.FORMAT_A8, w, h)
    ctx = cairo.Context(surface)

    # Apply CTM if present (before building path)
    if "ctm" in tc and tc["ctm"]:
        a, b, c, d, e, f = tc["ctm"]
        ctx.set_matrix(cairo.Matrix(a, b, c, d, e, f))

    # build path
    for seg in tc["path"]:
        cmd, pts = seg["cmd"], seg["pts"]
        if cmd == "M":
            ctx.move_to(pts[0][0], pts[0][1])
        elif cmd == "L":
            ctx.line_to(pts[0][0], pts[0][1])
        elif cmd == "Q":
            # Cairo doesn't have quadratic Bezier - convert to cubic
            # Q(p0, p1, p2) -> C(p0, p0 + 2/3*(p1-p0), p2 + 2/3*(p1-p2), p2)
            x0, y0 = ctx.get_current_point()
            x1, y1 = pts[0]
            x2, y2 = pts[1]
            ctx.curve_to(
                x0 + 2/3 * (x1 - x0), y0 + 2/3 * (y1 - y0),
                x2 + 2/3 * (x1 - x2), y2 + 2/3 * (y1 - y2),
                x2, y2
            )
        elif cmd == "C":
            ctx.curve_to(
                pts[0][0], pts[0][1],
                pts[1][0], pts[1][1],
                pts[2][0], pts[2][1]
            )
        elif cmd == "Z":
            ctx.close_path()

    # apply operation
    if tc["op"] == "fill":
        rule_map = {
            "nonzero": cairo.FILL_RULE_WINDING,
            "evenodd": cairo.FILL_RULE_EVEN_ODD,
        }
        ctx.set_fill_rule(rule_map[tc["fill_rule"]])
        ctx.set_source_rgba(1, 1, 1, 1)
        ctx.fill()
    else:  # stroke
        ctx.set_line_width(tc["line_width"])

        cap_map = {
            "butt": cairo.LINE_CAP_BUTT,
            "round": cairo.LINE_CAP_ROUND,
            "square": cairo.LINE_CAP_SQUARE,
        }
        ctx.set_line_cap(cap_map[tc["line_cap"]])

        join_map = {
            "miter": cairo.LINE_JOIN_MITER,
            "round": cairo.LINE_JOIN_ROUND,
            "bevel": cairo.LINE_JOIN_BEVEL,
        }
        ctx.set_line_join(join_map[tc["line_join"]])

        ctx.set_miter_limit(tc["miter_limit"])

        if tc.get("dash"):
            ctx.set_dash(tc["dash"], tc.get("dash_phase", 0))

        ctx.set_source_rgba(1, 1, 1, 1)
        ctx.stroke()

    surface.write_to_png(str(output_dir / f"{name}.png"))


if __name__ == "__main__":
    main()
