import { describe, expect, it } from 'vitest';
import {
  FILE_TRANSFER_MARGIN,
  fileTransferModalPosition,
  fileTransferModalSize,
} from './shell-model.js';

describe('fileTransferModalSize', () => {
  it('keeps the modal inside the viewport', () => {
    expect(fileTransferModalSize(1400, 900, 1200, 800)).toEqual({
      width: 1200 - FILE_TRANSFER_MARGIN * 2,
      height: 800 - FILE_TRANSFER_MARGIN * 2,
    });
  });

  it('uses the normal minimum size when space is available', () => {
    expect(fileTransferModalSize(300, 200, 1600, 1000)).toEqual({
      width: 720,
      height: 460,
    });
  });

  it('shrinks below the normal minimum on a small viewport', () => {
    expect(fileTransferModalSize(720, 460, 600, 400)).toEqual({
      width: 564,
      height: 364,
    });
  });
});

describe('fileTransferModalPosition', () => {
  it('clamps every edge to the modal margin', () => {
    expect(fileTransferModalPosition(-20, 900, 700, 500, 1200, 800)).toEqual({
      left: 18,
      top: 282,
    });
  });
});
