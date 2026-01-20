I want to write a new rasteriser from scratch.  This must work for the PDF and PostScript imaging models:
* curves are made up of straight line segments and quadratic or cubic Bezier curves.
* Fill rules are "nonzero winding number" and "even-odd".
* The main primitive is filling closed curves.  This must be efficient, both for large areas (page-size) and small areas (glyphs in a font).

I also need methods to draw lines with different styles applied.  Here I only need styles available in PDF:
* line width: >0
* line cap: "butt", "round", "square"
* line join: "miter", "round", "bevel"
* miter limit:
* dash pattern and phase
* dash phase
I imagine that these will be converted to outline curves to be filled, but if there is a better way, I'm also open to this.

I will deal with color separately, so I just need the amount of color here.

Instead of just getting pixels covered, I want to get a grayscale version: a number in [0.0, 1.0] which indicates which fraction of the pixel is covered by interior of the curve.  This should correspond to what I would get if I rendered the shape in the usual way (black if the pixel centre is inside, white otherwise) on a higher-resolution grid and then downsamples to the device grid, but without actually constructing the supersampled image.

Priorities are (1) correctness and (2) speed.
