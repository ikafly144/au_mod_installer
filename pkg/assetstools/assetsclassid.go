package assetstools

import "fmt"

type AssetsClassID int32

const (
	ClassID_Object                                            AssetsClassID = 0
	ClassID_GameObject                                        AssetsClassID = 1
	ClassID_Component                                         AssetsClassID = 2
	ClassID_LevelGameManager                                  AssetsClassID = 3
	ClassID_Transform                                         AssetsClassID = 4
	ClassID_TimeManager                                       AssetsClassID = 5
	ClassID_GlobalGameManager                                 AssetsClassID = 6
	ClassID_Behaviour                                         AssetsClassID = 8
	ClassID_GameManager                                       AssetsClassID = 9
	ClassID_AudioManager                                      AssetsClassID = 11
	ClassID_ParticleAnimator                                  AssetsClassID = 12
	ClassID_InputManager                                      AssetsClassID = 13
	ClassID_EllipsoidParticleEmitter                          AssetsClassID = 15
	ClassID_Pipeline                                          AssetsClassID = 17
	ClassID_EditorExtension                                   AssetsClassID = 18
	ClassID_Physics2DSettings                                 AssetsClassID = 19
	ClassID_Camera                                            AssetsClassID = 20
	ClassID_Material                                          AssetsClassID = 21
	ClassID_MeshRenderer                                      AssetsClassID = 23
	ClassID_Renderer                                          AssetsClassID = 25
	ClassID_ParticleRenderer                                  AssetsClassID = 26
	ClassID_Texture                                           AssetsClassID = 27
	ClassID_Texture2D                                         AssetsClassID = 28
	ClassID_OcclusionCullingSettings                          AssetsClassID = 29
	ClassID_GraphicsSettings                                  AssetsClassID = 30
	ClassID_MeshFilter                                        AssetsClassID = 33
	ClassID_OcclusionPortal                                   AssetsClassID = 41
	ClassID_Mesh                                              AssetsClassID = 43
	ClassID_Skybox                                            AssetsClassID = 45
	ClassID_QualitySettings                                   AssetsClassID = 47
	ClassID_Shader                                            AssetsClassID = 48
	ClassID_TextAsset                                         AssetsClassID = 49
	ClassID_Rigidbody2D                                       AssetsClassID = 50
	ClassID_Physics2DManager                                  AssetsClassID = 51
	ClassID_NotificationManager                               AssetsClassID = 52
	ClassID_Collider2D                                        AssetsClassID = 53
	ClassID_Rigidbody                                         AssetsClassID = 54
	ClassID_PhysicsManager                                    AssetsClassID = 55
	ClassID_Collider                                          AssetsClassID = 56
	ClassID_Joint                                             AssetsClassID = 57
	ClassID_CircleCollider2D                                  AssetsClassID = 58
	ClassID_HingeJoint                                        AssetsClassID = 59
	ClassID_PolygonCollider2D                                 AssetsClassID = 60
	ClassID_BoxCollider2D                                     AssetsClassID = 61
	ClassID_PhysicsMaterial2D                                 AssetsClassID = 62
	ClassID_MeshCollider                                      AssetsClassID = 64
	ClassID_BoxCollider                                       AssetsClassID = 65
	ClassID_CompositeCollider2D                               AssetsClassID = 66
	ClassID_EdgeCollider2D                                    AssetsClassID = 68
	ClassID_PolygonColliderBase2D                             AssetsClassID = 69
	ClassID_CapsuleCollider2D                                 AssetsClassID = 70
	ClassID_AnimationManager                                  AssetsClassID = 71
	ClassID_ComputeShader                                     AssetsClassID = 72
	ClassID_AnimationClip                                     AssetsClassID = 74
	ClassID_ConstantForce                                     AssetsClassID = 75
	ClassID_WorldParticleCollider                             AssetsClassID = 76
	ClassID_TagManager                                        AssetsClassID = 78
	ClassID_AudioListener                                     AssetsClassID = 81
	ClassID_AudioSource                                       AssetsClassID = 82
	ClassID_AudioClip                                         AssetsClassID = 83
	ClassID_RenderTexture                                     AssetsClassID = 84
	ClassID_CustomRenderTexture                               AssetsClassID = 86
	ClassID_MeshParticleEmitter                               AssetsClassID = 87
	ClassID_ParticleEmitter                                   AssetsClassID = 88
	ClassID_Cubemap                                           AssetsClassID = 89
	ClassID_Avatar                                            AssetsClassID = 90
	ClassID_AnimatorController                                AssetsClassID = 91
	ClassID_GUILayer                                          AssetsClassID = 92
	ClassID_RuntimeAnimatorController                         AssetsClassID = 93
	ClassID_ScriptMapper                                      AssetsClassID = 94
	ClassID_Animator                                          AssetsClassID = 95
	ClassID_TrailRenderer                                     AssetsClassID = 96
	ClassID_DelayedCallManager                                AssetsClassID = 98
	ClassID_TextMesh                                          AssetsClassID = 102
	ClassID_RenderSettings                                    AssetsClassID = 104
	ClassID_Light                                             AssetsClassID = 108
	ClassID_CGProgram                                         AssetsClassID = 109
	ClassID_BaseAnimationTrack                                AssetsClassID = 110
	ClassID_Animation                                         AssetsClassID = 111
	ClassID_MonoBehaviour                                     AssetsClassID = 114
	ClassID_MonoScript                                        AssetsClassID = 115
	ClassID_MonoManager                                       AssetsClassID = 116
	ClassID_Texture3D                                         AssetsClassID = 117
	ClassID_NewAnimationTrack                                 AssetsClassID = 118
	ClassID_Projector                                         AssetsClassID = 119
	ClassID_LineRenderer                                      AssetsClassID = 120
	ClassID_Flare                                             AssetsClassID = 121
	ClassID_Halo                                              AssetsClassID = 122
	ClassID_LensFlare                                         AssetsClassID = 123
	ClassID_FlareLayer                                        AssetsClassID = 124
	ClassID_HaloLayer                                         AssetsClassID = 125
	ClassID_NavMeshProjectSettings                            AssetsClassID = 126
	ClassID_HaloManager                                       AssetsClassID = 127
	ClassID_Font                                              AssetsClassID = 128
	ClassID_PlayerSettings                                    AssetsClassID = 129
	ClassID_NamedObject                                       AssetsClassID = 130
	ClassID_GUITexture                                        AssetsClassID = 131
	ClassID_GUIText                                           AssetsClassID = 132
	ClassID_GUIElement                                        AssetsClassID = 133
	ClassID_PhysicMaterial                                    AssetsClassID = 134
	ClassID_SphereCollider                                    AssetsClassID = 135
	ClassID_CapsuleCollider                                   AssetsClassID = 136
	ClassID_SkinnedMeshRenderer                               AssetsClassID = 137
	ClassID_FixedJoint                                        AssetsClassID = 138
	ClassID_RaycastCollider                                   AssetsClassID = 140
	ClassID_BuildSettings                                     AssetsClassID = 141
	ClassID_AssetBundle                                       AssetsClassID = 142
	ClassID_CharacterController                               AssetsClassID = 143
	ClassID_CharacterJoint                                    AssetsClassID = 144
	ClassID_SpringJoint                                       AssetsClassID = 145
	ClassID_WheelCollider                                     AssetsClassID = 146
	ClassID_ResourceManager                                   AssetsClassID = 147
	ClassID_NetworkView                                       AssetsClassID = 148
	ClassID_NetworkManager                                    AssetsClassID = 149
	ClassID_PreloadData                                       AssetsClassID = 150
	ClassID_MovieTexture                                      AssetsClassID = 152
	ClassID_ConfigurableJoint                                 AssetsClassID = 153
	ClassID_TerrainCollider                                   AssetsClassID = 154
	ClassID_MasterServerInterface                             AssetsClassID = 155
	ClassID_TerrainData                                       AssetsClassID = 156
	ClassID_LightmapSettings                                  AssetsClassID = 157
	ClassID_WebCamTexture                                     AssetsClassID = 158
	ClassID_EditorSettings                                    AssetsClassID = 159
	ClassID_InteractiveCloth                                  AssetsClassID = 160
	ClassID_ClothRenderer                                     AssetsClassID = 161
	ClassID_EditorUserSettings                                AssetsClassID = 162
	ClassID_SkinnedCloth                                      AssetsClassID = 163
	ClassID_AudioReverbFilter                                 AssetsClassID = 164
	ClassID_AudioHighPassFilter                               AssetsClassID = 165
	ClassID_AudioChorusFilter                                 AssetsClassID = 166
	ClassID_AudioReverbZone                                   AssetsClassID = 167
	ClassID_AudioEchoFilter                                   AssetsClassID = 168
	ClassID_AudioLowPassFilter                                AssetsClassID = 169
	ClassID_AudioDistortionFilter                             AssetsClassID = 170
	ClassID_SparseTexture                                     AssetsClassID = 171
	ClassID_AudioBehaviour                                    AssetsClassID = 180
	ClassID_AudioFilter                                       AssetsClassID = 181
	ClassID_WindZone                                          AssetsClassID = 182
	ClassID_Cloth                                             AssetsClassID = 183
	ClassID_SubstanceArchive                                  AssetsClassID = 184
	ClassID_ProceduralMaterial                                AssetsClassID = 185
	ClassID_ProceduralTexture                                 AssetsClassID = 186
	ClassID_Texture2DArray                                    AssetsClassID = 187
	ClassID_CubemapArray                                      AssetsClassID = 188
	ClassID_OffMeshLink                                       AssetsClassID = 191
	ClassID_OcclusionArea                                     AssetsClassID = 192
	ClassID_Tree                                              AssetsClassID = 193
	ClassID_NavMeshObsolete                                   AssetsClassID = 194
	ClassID_NavMeshAgent                                      AssetsClassID = 195
	ClassID_NavMeshSettings                                   AssetsClassID = 196
	ClassID_LightProbesLegacy                                 AssetsClassID = 197 // originally LightProbes
	ClassID_ParticleSystem                                    AssetsClassID = 198
	ClassID_ParticleSystemRenderer                            AssetsClassID = 199
	ClassID_ShaderVariantCollection                           AssetsClassID = 200
	ClassID_LODGroup                                          AssetsClassID = 205
	ClassID_BlendTree                                         AssetsClassID = 206
	ClassID_Motion                                            AssetsClassID = 207
	ClassID_NavMeshObstacle                                   AssetsClassID = 208
	ClassID_SortingGroup                                      AssetsClassID = 210
	ClassID_SpriteRenderer                                    AssetsClassID = 212
	ClassID_Sprite                                            AssetsClassID = 213
	ClassID_CachedSpriteAtlas                                 AssetsClassID = 214
	ClassID_ReflectionProbe                                   AssetsClassID = 215
	ClassID_ReflectionProbes                                  AssetsClassID = 216
	ClassID_Terrain                                           AssetsClassID = 218
	ClassID_LightProbeGroup                                   AssetsClassID = 220
	ClassID_AnimatorOverrideController                        AssetsClassID = 221
	ClassID_CanvasRenderer                                    AssetsClassID = 222
	ClassID_Canvas                                            AssetsClassID = 223
	ClassID_RectTransform                                     AssetsClassID = 224
	ClassID_CanvasGroup                                       AssetsClassID = 225
	ClassID_BillboardAsset                                    AssetsClassID = 226
	ClassID_BillboardRenderer                                 AssetsClassID = 227
	ClassID_SpeedTreeWindAsset                                AssetsClassID = 228
	ClassID_AnchoredJoint2D                                   AssetsClassID = 229
	ClassID_Joint2D                                           AssetsClassID = 230
	ClassID_SpringJoint2D                                     AssetsClassID = 231
	ClassID_DistanceJoint2D                                   AssetsClassID = 232
	ClassID_HingeJoint2D                                      AssetsClassID = 233
	ClassID_SliderJoint2D                                     AssetsClassID = 234
	ClassID_WheelJoint2D                                      AssetsClassID = 235
	ClassID_ClusterInputManager                               AssetsClassID = 236
	ClassID_BaseVideoTexture                                  AssetsClassID = 237
	ClassID_NavMeshData                                       AssetsClassID = 238
	ClassID_AudioMixer                                        AssetsClassID = 240
	ClassID_AudioMixerController                              AssetsClassID = 241
	ClassID_AudioMixerGroupController                         AssetsClassID = 243
	ClassID_AudioMixerEffectController                        AssetsClassID = 244
	ClassID_AudioMixerSnapshotController                      AssetsClassID = 245
	ClassID_PhysicsUpdateBehaviour2D                          AssetsClassID = 246
	ClassID_ConstantForce2D                                   AssetsClassID = 247
	ClassID_Effector2D                                        AssetsClassID = 248
	ClassID_AreaEffector2D                                    AssetsClassID = 249
	ClassID_PointEffector2D                                   AssetsClassID = 250
	ClassID_PlatformEffector2D                                AssetsClassID = 251
	ClassID_SurfaceEffector2D                                 AssetsClassID = 252
	ClassID_BuoyancyEffector2D                                AssetsClassID = 253
	ClassID_RelativeJoint2D                                   AssetsClassID = 254
	ClassID_FixedJoint2D                                      AssetsClassID = 255
	ClassID_FrictionJoint2D                                   AssetsClassID = 256
	ClassID_TargetJoint2D                                     AssetsClassID = 257
	ClassID_LightProbes                                       AssetsClassID = 258
	ClassID_LightProbeProxyVolume                             AssetsClassID = 259
	ClassID_SampleClip                                        AssetsClassID = 271
	ClassID_AudioMixerSnapshot                                AssetsClassID = 272
	ClassID_AudioMixerGroup                                   AssetsClassID = 273
	ClassID_NScreenBridge                                     AssetsClassID = 280
	ClassID_AssetBundleManifest                               AssetsClassID = 290
	ClassID_UnityAdsManager                                   AssetsClassID = 292
	ClassID_RuntimeInitializeOnLoadManager                    AssetsClassID = 300
	ClassID_CloudWebServicesManager                           AssetsClassID = 301
	ClassID_CloudServiceHandlerBehaviour                      AssetsClassID = 302
	ClassID_UnityAnalyticsManager                             AssetsClassID = 303
	ClassID_CrashReportManager                                AssetsClassID = 304
	ClassID_PerformanceReportingManager                       AssetsClassID = 305
	ClassID_UnityConnectSettings                              AssetsClassID = 310
	ClassID_AvatarMask                                        AssetsClassID = 319
	ClassID_PlayableDirector                                  AssetsClassID = 320
	ClassID_VideoPlayer                                       AssetsClassID = 328
	ClassID_VideoClip                                         AssetsClassID = 329
	ClassID_ParticleSystemForceField                          AssetsClassID = 330
	ClassID_SpriteMask                                        AssetsClassID = 331
	ClassID_WorldAnchor                                       AssetsClassID = 362
	ClassID_OcclusionCullingData                              AssetsClassID = 363
	ClassID_SmallestEditorClassID                             AssetsClassID = 1000
	ClassID_PrefabInstance                                    AssetsClassID = 1001
	ClassID_EditorExtensionImpl                               AssetsClassID = 1002
	ClassID_AssetImporter                                     AssetsClassID = 1003
	ClassID_AssetDatabaseV1                                   AssetsClassID = 1004
	ClassID_Mesh3DSImporter                                   AssetsClassID = 1005
	ClassID_TextureImporter                                   AssetsClassID = 1006
	ClassID_ShaderImporter                                    AssetsClassID = 1007
	ClassID_ComputeShaderImporter                             AssetsClassID = 1008
	ClassID_AudioImporter                                     AssetsClassID = 1020
	ClassID_HierarchyState                                    AssetsClassID = 1026
	ClassID_GUIDSerializer                                    AssetsClassID = 1027
	ClassID_AssetMetaData                                     AssetsClassID = 1028
	ClassID_DefaultAsset                                      AssetsClassID = 1029
	ClassID_DefaultImporter                                   AssetsClassID = 1030
	ClassID_TextScriptImporter                                AssetsClassID = 1031
	ClassID_SceneAsset                                        AssetsClassID = 1032
	ClassID_NativeFormatImporter                              AssetsClassID = 1034
	ClassID_MonoImporter                                      AssetsClassID = 1035
	ClassID_AssetServerCache                                  AssetsClassID = 1037
	ClassID_LibraryAssetImporter                              AssetsClassID = 1038
	ClassID_ModelImporter                                     AssetsClassID = 1040
	ClassID_FBXImporter                                       AssetsClassID = 1041
	ClassID_TrueTypeFontImporter                              AssetsClassID = 1042
	ClassID_MovieImporter                                     AssetsClassID = 1044
	ClassID_EditorBuildSettings                               AssetsClassID = 1045
	ClassID_DDSImporter                                       AssetsClassID = 1046
	ClassID_InspectorExpandedState                            AssetsClassID = 1048
	ClassID_AnnotationManager                                 AssetsClassID = 1049
	ClassID_PluginImporter                                    AssetsClassID = 1050
	ClassID_EditorUserBuildSettings                           AssetsClassID = 1051
	ClassID_PVRImporter                                       AssetsClassID = 1052
	ClassID_ASTCImporter                                      AssetsClassID = 1053
	ClassID_KTXImporter                                       AssetsClassID = 1054
	ClassID_IHVImageFormatImporter                            AssetsClassID = 1055
	ClassID_AnimatorStateTransition                           AssetsClassID = 1101
	ClassID_AnimatorState                                     AssetsClassID = 1102
	ClassID_HumanTemplate                                     AssetsClassID = 1105
	ClassID_AnimatorStateMachine                              AssetsClassID = 1107
	ClassID_PreviewAnimationClip                              AssetsClassID = 1108
	ClassID_AnimatorTransition                                AssetsClassID = 1109
	ClassID_SpeedTreeImporter                                 AssetsClassID = 1110
	ClassID_AnimatorTransitionBase                            AssetsClassID = 1111
	ClassID_SubstanceImporter                                 AssetsClassID = 1112
	ClassID_LightmapParameters                                AssetsClassID = 1113
	ClassID_LightingDataAsset                                 AssetsClassID = 1120
	ClassID_GISRaster                                         AssetsClassID = 1121
	ClassID_GISRasterImporter                                 AssetsClassID = 1122
	ClassID_CadImporter                                       AssetsClassID = 1123
	ClassID_SketchUpImporter                                  AssetsClassID = 1124
	ClassID_BuildReport                                       AssetsClassID = 1125
	ClassID_PackedAssets                                      AssetsClassID = 1126
	ClassID_VideoClipImporter                                 AssetsClassID = 1127
	ClassID_ActivationLogComponent                            AssetsClassID = 2000
	ClassID_int                                               AssetsClassID = 100000
	ClassID_bool                                              AssetsClassID = 100001
	ClassID_float                                             AssetsClassID = 100002
	ClassID_MonoObject                                        AssetsClassID = 100003
	ClassID_Collision                                         AssetsClassID = 100004
	ClassID_Vector3f                                          AssetsClassID = 100005
	ClassID_RootMotionData                                    AssetsClassID = 100006
	ClassID_Collision2D                                       AssetsClassID = 100007
	ClassID_AudioMixerLiveUpdateFloat                         AssetsClassID = 100008
	ClassID_AudioMixerLiveUpdateBool                          AssetsClassID = 100009
	ClassID_Polygon2D                                         AssetsClassID = 100010
	ClassID_void                                              AssetsClassID = 100011
	ClassID_TilemapCollider2D                                 AssetsClassID = 19719996
	ClassID_AssetImporterLog                                  AssetsClassID = 41386430
	ClassID_VFXRenderer                                       AssetsClassID = 73398921
	ClassID_SerializableManagedRefTestClass                   AssetsClassID = 76251197
	ClassID_Grid                                              AssetsClassID = 156049354
	ClassID_ScenesUsingAssets                                 AssetsClassID = 156483287
	ClassID_ArticulationBody                                  AssetsClassID = 171741748
	ClassID_Preset                                            AssetsClassID = 181963792
	ClassID_EmptyObject                                       AssetsClassID = 277625683
	ClassID_IConstraint                                       AssetsClassID = 285090594
	ClassID_TestObjectWithSpecialLayoutOne                    AssetsClassID = 293259124
	ClassID_AssemblyDefinitionReferenceImporter               AssetsClassID = 294290339
	ClassID_SiblingDerived                                    AssetsClassID = 334799969
	ClassID_TestObjectWithSerializedMapStringNonAlignedStruct AssetsClassID = 342846651
	ClassID_SubDerived                                        AssetsClassID = 367388927
	ClassID_AssetImportInProgressProxy                        AssetsClassID = 369655926
	ClassID_PluginBuildInfo                                   AssetsClassID = 382020655
	ClassID_EditorProjectAccess                               AssetsClassID = 426301858
	ClassID_PrefabImporter                                    AssetsClassID = 468431735
	ClassID_TestObjectWithSerializedArray                     AssetsClassID = 478637458
	ClassID_TestObjectWithSerializedAnimationCurve            AssetsClassID = 478637459
	ClassID_TilemapRenderer                                   AssetsClassID = 483693784
	ClassID_ScriptableCamera                                  AssetsClassID = 488575907
	ClassID_SpriteAtlasAsset                                  AssetsClassID = 612988286
	ClassID_SpriteAtlasDatabase                               AssetsClassID = 638013454
	ClassID_AudioBuildInfo                                    AssetsClassID = 641289076
	ClassID_CachedSpriteAtlasRuntimeData                      AssetsClassID = 644342135
	ClassID_RendererFake                                      AssetsClassID = 646504946
	ClassID_AssemblyDefinitionReferenceAsset                  AssetsClassID = 662584278
	ClassID_BuiltAssetBundleInfoSet                           AssetsClassID = 668709126
	ClassID_SpriteAtlas                                       AssetsClassID = 687078895
	ClassID_RayTracingShaderImporter                          AssetsClassID = 747330370
	ClassID_PreviewImporter                                   AssetsClassID = 815301076
	ClassID_RayTracingShader                                  AssetsClassID = 825902497
	ClassID_LightingSettings                                  AssetsClassID = 850595691
	ClassID_PlatformModuleSetup                               AssetsClassID = 877146078
	ClassID_VersionControlSettings                            AssetsClassID = 890905787
	ClassID_AimConstraint                                     AssetsClassID = 895512359
	ClassID_VFXManager                                        AssetsClassID = 937362698
	ClassID_VisualEffectSubgraph                              AssetsClassID = 994735392
	ClassID_RuleSetFileAsset                                  AssetsClassID = 954905827
	ClassID_VisualEffectSubgraphOperator                      AssetsClassID = 994735403
	ClassID_VisualEffectSubgraphBlock                         AssetsClassID = 994735404
	ClassID_Prefab                                            AssetsClassID = 1001480554
	ClassID_LocalizationImporter                              AssetsClassID = 1027052791
	ClassID_Derived                                           AssetsClassID = 1091556383
	ClassID_PropertyModificationsTargetTestObject             AssetsClassID = 1111377672
	ClassID_ReferencesArtifactGenerator                       AssetsClassID = 1114811875
	ClassID_AssemblyDefinitionAsset                           AssetsClassID = 1152215463
	ClassID_SceneVisibilityState                              AssetsClassID = 1154873562
	ClassID_LookAtConstraint                                  AssetsClassID = 1183024399
	ClassID_SpriteAtlasImporter                               AssetsClassID = 1210832254
	ClassID_MultiArtifactTestImporter                         AssetsClassID = 1223240404
	ClassID_GameObjectRecorder                                AssetsClassID = 1268269756
	ClassID_LightingDataAssetParent                           AssetsClassID = 1325145578
	ClassID_PresetManager                                     AssetsClassID = 1386491679
	ClassID_TestObjectWithSpecialLayoutTwo                    AssetsClassID = 1392443030
	ClassID_StreamingManager                                  AssetsClassID = 1403656975
	ClassID_LowerResBlitTexture                               AssetsClassID = 1480428607
	ClassID_StreamingController                               AssetsClassID = 1542919678
	ClassID_RenderPassAttachment                              AssetsClassID = 1571458007
	ClassID_TestObjectVectorPairStringBool                    AssetsClassID = 1628831178
	ClassID_GridLayout                                        AssetsClassID = 1742807556
	ClassID_AssemblyDefinitionImporter                        AssetsClassID = 1766753193
	ClassID_ParentConstraint                                  AssetsClassID = 1773428102
	ClassID_FakeComponent                                     AssetsClassID = 1803986026
	ClassID_RuleSetFileImporter                               AssetsClassID = 1777034230
	ClassID_PositionConstraint                                AssetsClassID = 1818360608
	ClassID_RotationConstraint                                AssetsClassID = 1818360609
	ClassID_ScaleConstraint                                   AssetsClassID = 1818360610
	ClassID_Tilemap                                           AssetsClassID = 1839735485
	ClassID_PackageManifest                                   AssetsClassID = 1896753125
	ClassID_PackageManifestImporter                           AssetsClassID = 1896753126
	ClassID_TerrainLayer                                      AssetsClassID = 1953259897
	ClassID_SpriteShapeRenderer                               AssetsClassID = 1971053207
	ClassID_NativeObjectType                                  AssetsClassID = 1977754360
	ClassID_TestObjectWithSerializedMapStringBool             AssetsClassID = 1981279845
	ClassID_SerializableManagedHost                           AssetsClassID = 1995898324
	ClassID_VisualEffectAsset                                 AssetsClassID = 2058629509
	ClassID_VisualEffectImporter                              AssetsClassID = 2058629510
	ClassID_VisualEffectResource                              AssetsClassID = 2058629511
	ClassID_VisualEffectObject                                AssetsClassID = 2059678085
	ClassID_VisualEffect                                      AssetsClassID = 2083052967
	ClassID_LocalizationAsset                                 AssetsClassID = 2083778819
	ClassID_ScriptedImporter                                  AssetsClassID = 2089858483
	ClassID_TilemapEditorUserSettings                         AssetsClassID = 2126867596
)

var AssetsClassIDName = map[AssetsClassID]string{
	ClassID_Object:                              "Object",
	ClassID_GameObject:                          "GameObject",
	ClassID_Component:                           "Component",
	ClassID_LevelGameManager:                    "LevelGameManager",
	ClassID_Transform:                           "Transform",
	ClassID_TimeManager:                         "TimeManager",
	ClassID_GlobalGameManager:                   "GlobalGameManager",
	ClassID_Behaviour:                           "Behaviour",
	ClassID_GameManager:                         "GameManager",
	ClassID_AudioManager:                        "AudioManager",
	ClassID_ParticleAnimator:                    "ParticleAnimator",
	ClassID_InputManager:                        "InputManager",
	ClassID_EllipsoidParticleEmitter:            "EllipsoidParticleEmitter",
	ClassID_Pipeline:                            "Pipeline",
	ClassID_EditorExtension:                     "EditorExtension",
	ClassID_Physics2DSettings:                   "Physics2DSettings",
	ClassID_Camera:                              "Camera",
	ClassID_Material:                            "Material",
	ClassID_MeshRenderer:                        "MeshRenderer",
	ClassID_Renderer:                            "Renderer",
	ClassID_ParticleRenderer:                    "ParticleRenderer",
	ClassID_Texture:                             "Texture",
	ClassID_Texture2D:                           "Texture2D",
	ClassID_OcclusionCullingSettings:            "OcclusionCullingSettings",
	ClassID_GraphicsSettings:                    "GraphicsSettings",
	ClassID_MeshFilter:                          "MeshFilter",
	ClassID_OcclusionPortal:                     "OcclusionPortal",
	ClassID_Mesh:                                "Mesh",
	ClassID_Skybox:                              "Skybox",
	ClassID_QualitySettings:                     "QualitySettings",
	ClassID_Shader:                              "Shader",
	ClassID_TextAsset:                           "TextAsset",
	ClassID_Rigidbody2D:                         "Rigidbody2D",
	ClassID_Physics2DManager:                    "Physics2DManager",
	ClassID_NotificationManager:                 "NotificationManager",
	ClassID_Collider2D:                          "Collider2D",
	ClassID_Rigidbody:                           "Rigidbody",
	ClassID_PhysicsManager:                      "PhysicsManager",
	ClassID_Collider:                            "Collider",
	ClassID_Joint:                               "Joint",
	ClassID_CircleCollider2D:                    "CircleCollider2D",
	ClassID_HingeJoint:                          "HingeJoint",
	ClassID_PolygonCollider2D:                   "PolygonCollider2D",
	ClassID_BoxCollider2D:                       "BoxCollider2D",
	ClassID_PhysicsMaterial2D:                   "PhysicsMaterial2D",
	ClassID_MeshCollider:                        "MeshCollider",
	ClassID_BoxCollider:                         "BoxCollider",
	ClassID_CompositeCollider2D:                 "CompositeCollider2D",
	ClassID_EdgeCollider2D:                      "EdgeCollider2D",
	ClassID_PolygonColliderBase2D:               "PolygonColliderBase2D",
	ClassID_CapsuleCollider2D:                   "CapsuleCollider2D",
	ClassID_AnimationManager:                    "AnimationManager",
	ClassID_ComputeShader:                       "ComputeShader",
	ClassID_AnimationClip:                       "AnimationClip",
	ClassID_ConstantForce:                       "ConstantForce",
	ClassID_WorldParticleCollider:               "WorldParticleCollider",
	ClassID_TagManager:                          "TagManager",
	ClassID_AudioListener:                       "AudioListener",
	ClassID_AudioSource:                         "AudioSource",
	ClassID_AudioClip:                           "AudioClip",
	ClassID_RenderTexture:                       "RenderTexture",
	ClassID_CustomRenderTexture:                 "CustomRenderTexture",
	ClassID_MeshParticleEmitter:                 "MeshParticleEmitter",
	ClassID_ParticleEmitter:                     "ParticleEmitter",
	ClassID_Cubemap:                             "Cubemap",
	ClassID_Avatar:                              "Avatar",
	ClassID_AnimatorController:                  "AnimatorController",
	ClassID_GUILayer:                            "GUILayer",
	ClassID_RuntimeAnimatorController:           "RuntimeAnimatorController",
	ClassID_ScriptMapper:                        "ScriptMapper",
	ClassID_Animator:                            "Animator",
	ClassID_TrailRenderer:                       "TrailRenderer",
	ClassID_DelayedCallManager:                  "DelayedCallManager",
	ClassID_TextMesh:                            "TextMesh",
	ClassID_RenderSettings:                      "RenderSettings",
	ClassID_Light:                               "Light",
	ClassID_CGProgram:                           "CGProgram",
	ClassID_BaseAnimationTrack:                  "BaseAnimationTrack",
	ClassID_Animation:                           "Animation",
	ClassID_MonoBehaviour:                       "MonoBehaviour",
	ClassID_MonoScript:                          "MonoScript",
	ClassID_MonoManager:                         "MonoManager",
	ClassID_Texture3D:                           "Texture3D",
	ClassID_NewAnimationTrack:                   "NewAnimationTrack",
	ClassID_Projector:                           "Projector",
	ClassID_LineRenderer:                        "LineRenderer",
	ClassID_Flare:                               "Flare",
	ClassID_Halo:                                "Halo",
	ClassID_LensFlare:                           "LensFlare",
	ClassID_FlareLayer:                          "FlareLayer",
	ClassID_HaloLayer:                           "HaloLayer",
	ClassID_NavMeshProjectSettings:              "NavMeshProjectSettings",
	ClassID_HaloManager:                         "HaloManager",
	ClassID_Font:                                "Font",
	ClassID_PlayerSettings:                      "PlayerSettings",
	ClassID_NamedObject:                         "NamedObject",
	ClassID_GUITexture:                          "GUITexture",
	ClassID_GUIText:                             "GUIText",
	ClassID_GUIElement:                          "GUIElement",
	ClassID_PhysicMaterial:                      "PhysicMaterial",
	ClassID_SphereCollider:                      "SphereCollider",
	ClassID_CapsuleCollider:                     "CapsuleCollider",
	ClassID_SkinnedMeshRenderer:                 "SkinnedMeshRenderer",
	ClassID_FixedJoint:                          "FixedJoint",
	ClassID_RaycastCollider:                     "RaycastCollider",
	ClassID_BuildSettings:                       "BuildSettings",
	ClassID_AssetBundle:                         "AssetBundle",
	ClassID_CharacterController:                 "CharacterController",
	ClassID_CharacterJoint:                      "CharacterJoint",
	ClassID_SpringJoint:                         "SpringJoint",
	ClassID_WheelCollider:                       "WheelCollider",
	ClassID_ResourceManager:                     "ResourceManager",
	ClassID_NetworkView:                         "NetworkView",
	ClassID_NetworkManager:                      "NetworkManager",
	ClassID_PreloadData:                         "PreloadData",
	ClassID_MovieTexture:                        "MovieTexture",
	ClassID_ConfigurableJoint:                   "ConfigurableJoint",
	ClassID_TerrainCollider:                     "TerrainCollider",
	ClassID_MasterServerInterface:               "MasterServerInterface",
	ClassID_TerrainData:                         "TerrainData",
	ClassID_LightmapSettings:                    "LightmapSettings",
	ClassID_WebCamTexture:                       "WebCamTexture",
	ClassID_EditorSettings:                      "EditorSettings",
	ClassID_InteractiveCloth:                    "InteractiveCloth",
	ClassID_ClothRenderer:                       "ClothRenderer",
	ClassID_EditorUserSettings:                  "EditorUserSettings",
	ClassID_SkinnedCloth:                        "SkinnedCloth",
	ClassID_AudioReverbFilter:                   "AudioReverbFilter",
	ClassID_AudioHighPassFilter:                 "AudioHighPassFilter",
	ClassID_AudioChorusFilter:                   "AudioChorusFilter",
	ClassID_AudioReverbZone:                     "AudioReverbZone",
	ClassID_AudioEchoFilter:                     "AudioEchoFilter",
	ClassID_AudioLowPassFilter:                  "AudioLowPassFilter",
	ClassID_AudioDistortionFilter:               "AudioDistortionFilter",
	ClassID_SparseTexture:                       "SparseTexture",
	ClassID_AudioBehaviour:                      "AudioBehaviour",
	ClassID_AudioFilter:                         "AudioFilter",
	ClassID_WindZone:                            "WindZone",
	ClassID_Cloth:                               "Cloth",
	ClassID_SubstanceArchive:                    "SubstanceArchive",
	ClassID_ProceduralMaterial:                  "ProceduralMaterial",
	ClassID_ProceduralTexture:                   "ProceduralTexture",
	ClassID_Texture2DArray:                      "Texture2DArray",
	ClassID_CubemapArray:                        "CubemapArray",
	ClassID_OffMeshLink:                         "OffMeshLink",
	ClassID_OcclusionArea:                       "OcclusionArea",
	ClassID_Tree:                                "Tree",
	ClassID_NavMeshObsolete:                     "NavMeshObsolete",
	ClassID_NavMeshAgent:                        "NavMeshAgent",
	ClassID_NavMeshSettings:                     "NavMeshSettings",
	ClassID_LightProbesLegacy:                   "LightProbesLegacy",
	ClassID_ParticleSystem:                      "ParticleSystem",
	ClassID_ParticleSystemRenderer:              "ParticleSystemRenderer",
	ClassID_ShaderVariantCollection:             "ShaderVariantCollection",
	ClassID_LODGroup:                            "LODGroup",
	ClassID_BlendTree:                           "BlendTree",
	ClassID_Motion:                              "Motion",
	ClassID_NavMeshObstacle:                     "NavMeshObstacle",
	ClassID_SortingGroup:                        "SortingGroup",
	ClassID_SpriteRenderer:                      "SpriteRenderer",
	ClassID_Sprite:                              "Sprite",
	ClassID_CachedSpriteAtlas:                   "CachedSpriteAtlas",
	ClassID_ReflectionProbe:                     "ReflectionProbe",
	ClassID_ReflectionProbes:                    "ReflectionProbes",
	ClassID_Terrain:                             "Terrain",
	ClassID_LightProbeGroup:                     "LightProbeGroup",
	ClassID_AnimatorOverrideController:          "AnimatorOverrideController",
	ClassID_CanvasRenderer:                      "CanvasRenderer",
	ClassID_Canvas:                              "Canvas",
	ClassID_RectTransform:                       "RectTransform",
	ClassID_CanvasGroup:                         "CanvasGroup",
	ClassID_BillboardAsset:                      "BillboardAsset",
	ClassID_BillboardRenderer:                   "BillboardRenderer",
	ClassID_SpeedTreeWindAsset:                  "SpeedTreeWindAsset",
	ClassID_AnchoredJoint2D:                     "AnchoredJoint2D",
	ClassID_Joint2D:                             "Joint2D",
	ClassID_SpringJoint2D:                       "SpringJoint2D",
	ClassID_DistanceJoint2D:                     "DistanceJoint2D",
	ClassID_HingeJoint2D:                        "HingeJoint2D",
	ClassID_SliderJoint2D:                       "SliderJoint2D",
	ClassID_WheelJoint2D:                        "WheelJoint2D",
	ClassID_ClusterInputManager:                 "ClusterInputManager",
	ClassID_BaseVideoTexture:                    "BaseVideoTexture",
	ClassID_NavMeshData:                         "NavMeshData",
	ClassID_AudioMixer:                          "AudioMixer",
	ClassID_AudioMixerController:                "AudioMixerController",
	ClassID_AudioMixerGroupController:           "AudioMixerGroupController",
	ClassID_AudioMixerEffectController:          "AudioMixerEffectController",
	ClassID_AudioMixerSnapshotController:        "AudioMixerSnapshotController",
	ClassID_PhysicsUpdateBehaviour2D:            "PhysicsUpdateBehaviour2D",
	ClassID_ConstantForce2D:                     "ConstantForce2D",
	ClassID_Effector2D:                          "Effector2D",
	ClassID_AreaEffector2D:                      "AreaEffector2D",
	ClassID_PointEffector2D:                     "PointEffector2D",
	ClassID_PlatformEffector2D:                  "PlatformEffector2D",
	ClassID_SurfaceEffector2D:                   "SurfaceEffector2D",
	ClassID_BuoyancyEffector2D:                  "BuoyancyEffector2D",
	ClassID_RelativeJoint2D:                     "RelativeJoint2D",
	ClassID_FixedJoint2D:                        "FixedJoint2D",
	ClassID_FrictionJoint2D:                     "FrictionJoint2D",
	ClassID_TargetJoint2D:                       "TargetJoint2D",
	ClassID_LightProbes:                         "LightProbes",
	ClassID_LightProbeProxyVolume:               "LightProbeProxyVolume",
	ClassID_SampleClip:                          "SampleClip",
	ClassID_AudioMixerSnapshot:                  "AudioMixerSnapshot",
	ClassID_AudioMixerGroup:                     "AudioMixerGroup",
	ClassID_NScreenBridge:                       "NScreenBridge",
	ClassID_AssetBundleManifest:                 "AssetBundleManifest",
	ClassID_UnityAdsManager:                     "UnityAdsManager",
	ClassID_RuntimeInitializeOnLoadManager:      "RuntimeInitializeOnLoadManager",
	ClassID_CloudWebServicesManager:             "CloudWebServicesManager",
	ClassID_CloudServiceHandlerBehaviour:        "CloudServiceHandlerBehaviour",
	ClassID_UnityAnalyticsManager:               "UnityAnalyticsManager",
	ClassID_CrashReportManager:                  "CrashReportManager",
	ClassID_PerformanceReportingManager:         "PerformanceReportingManager",
	ClassID_UnityConnectSettings:                "UnityConnectSettings",
	ClassID_AvatarMask:                          "AvatarMask",
	ClassID_PlayableDirector:                    "PlayableDirector",
	ClassID_VideoPlayer:                         "VideoPlayer",
	ClassID_VideoClip:                           "VideoClip",
	ClassID_ParticleSystemForceField:            "ParticleSystemForceField",
	ClassID_SpriteMask:                          "SpriteMask",
	ClassID_WorldAnchor:                         "WorldAnchor",
	ClassID_OcclusionCullingData:                "OcclusionCullingData",
	ClassID_SmallestEditorClassID:               "SmallestEditorClassID",
	ClassID_PrefabInstance:                      "PrefabInstance",
	ClassID_EditorExtensionImpl:                 "EditorExtensionImpl",
	ClassID_AssetImporter:                       "AssetImporter",
	ClassID_AssetDatabaseV1:                     "AssetDatabaseV1",
	ClassID_Mesh3DSImporter:                     "Mesh3DSImporter",
	ClassID_TextureImporter:                     "TextureImporter",
	ClassID_ShaderImporter:                      "ShaderImporter",
	ClassID_ComputeShaderImporter:               "ComputeShaderImporter",
	ClassID_AudioImporter:                       "AudioImporter",
	ClassID_HierarchyState:                      "HierarchyState",
	ClassID_GUIDSerializer:                      "GUIDSerializer",
	ClassID_AssetMetaData:                       "AssetMetaData",
	ClassID_DefaultAsset:                        "DefaultAsset",
	ClassID_DefaultImporter:                     "DefaultImporter",
	ClassID_TextScriptImporter:                  "TextScriptImporter",
	ClassID_SceneAsset:                          "SceneAsset",
	ClassID_NativeFormatImporter:                "NativeFormatImporter",
	ClassID_MonoImporter:                        "MonoImporter",
	ClassID_AssetServerCache:                    "AssetServerCache",
	ClassID_LibraryAssetImporter:                "LibraryAssetImporter",
	ClassID_ModelImporter:                       "ModelImporter",
	ClassID_FBXImporter:                         "FBXImporter",
	ClassID_TrueTypeFontImporter:                "TrueTypeFontImporter",
	ClassID_MovieImporter:                       "MovieImporter",
	ClassID_EditorBuildSettings:                 "EditorBuildSettings",
	ClassID_DDSImporter:                         "DDSImporter",
	ClassID_InspectorExpandedState:              "InspectorExpandedState",
	ClassID_AnnotationManager:                   "AnnotationManager",
	ClassID_PluginImporter:                      "PluginImporter",
	ClassID_EditorUserBuildSettings:             "EditorUserBuildSettings",
	ClassID_PVRImporter:                         "PVRImporter",
	ClassID_ASTCImporter:                        "ASTCImporter",
	ClassID_KTXImporter:                         "KTXImporter",
	ClassID_IHVImageFormatImporter:              "IHVImageFormatImporter",
	ClassID_AnimatorStateTransition:             "AnimatorStateTransition",
	ClassID_AnimatorState:                       "AnimatorState",
	ClassID_HumanTemplate:                       "HumanTemplate",
	ClassID_AnimatorStateMachine:                "AnimatorStateMachine",
	ClassID_PreviewAnimationClip:                "PreviewAnimationClip",
	ClassID_AnimatorTransition:                  "AnimatorTransition",
	ClassID_SpeedTreeImporter:                   "SpeedTreeImporter",
	ClassID_AnimatorTransitionBase:              "AnimatorTransitionBase",
	ClassID_SubstanceImporter:                   "SubstanceImporter",
	ClassID_LightmapParameters:                  "LightmapParameters",
	ClassID_LightingDataAsset:                   "LightingDataAsset",
	ClassID_GISRaster:                           "GISRaster",
	ClassID_GISRasterImporter:                   "GISRasterImporter",
	ClassID_CadImporter:                         "CadImporter",
	ClassID_SketchUpImporter:                    "SketchUpImporter",
	ClassID_BuildReport:                         "BuildReport",
	ClassID_PackedAssets:                        "PackedAssets",
	ClassID_VideoClipImporter:                   "VideoClipImporter",
	ClassID_ActivationLogComponent:              "ActivationLogComponent",
	ClassID_int:                                 "int",
	ClassID_bool:                                "bool",
	ClassID_float:                               "float",
	ClassID_MonoObject:                          "MonoObject",
	ClassID_Collision:                           "Collision",
	ClassID_Vector3f:                            "Vector3f",
	ClassID_RootMotionData:                      "RootMotionData",
	ClassID_Collision2D:                         "Collision2D",
	ClassID_AudioMixerLiveUpdateFloat:           "AudioMixerLiveUpdateFloat",
	ClassID_AudioMixerLiveUpdateBool:            "AudioMixerLiveUpdateBool",
	ClassID_Polygon2D:                           "Polygon2D",
	ClassID_void:                                "void",
	ClassID_TilemapCollider2D:                   "TilemapCollider2D",
	ClassID_AssetImporterLog:                    "AssetImporterLog",
	ClassID_VFXRenderer:                         "VFXRenderer",
	ClassID_SerializableManagedRefTestClass:     "SerializableManagedRefTestClass",
	ClassID_Grid:                                "Grid",
	ClassID_ScenesUsingAssets:                   "ScenesUsingAssets",
	ClassID_ArticulationBody:                    "ArticulationBody",
	ClassID_Preset:                              "Preset",
	ClassID_EmptyObject:                         "EmptyObject",
	ClassID_IConstraint:                         "IConstraint",
	ClassID_TestObjectWithSpecialLayoutOne:      "TestObjectWithSpecialLayoutOne",
	ClassID_AssemblyDefinitionReferenceImporter: "AssemblyDefinitionReferenceImporter",
	ClassID_SiblingDerived:                      "SiblingDerived",
	ClassID_TestObjectWithSerializedMapStringNonAlignedStruct: "TestObjectWithSerializedMapStringNonAlignedStruct",
	ClassID_SubDerived:                             "SubDerived",
	ClassID_AssetImportInProgressProxy:             "AssetImportInProgressProxy",
	ClassID_PluginBuildInfo:                        "PluginBuildInfo",
	ClassID_EditorProjectAccess:                    "EditorProjectAccess",
	ClassID_PrefabImporter:                         "PrefabImporter",
	ClassID_TestObjectWithSerializedArray:          "TestObjectWithSerializedArray",
	ClassID_TestObjectWithSerializedAnimationCurve: "TestObjectWithSerializedAnimationCurve",
	ClassID_TilemapRenderer:                        "TilemapRenderer",
	ClassID_ScriptableCamera:                       "ScriptableCamera",
	ClassID_SpriteAtlasAsset:                       "SpriteAtlasAsset",
	ClassID_SpriteAtlasDatabase:                    "SpriteAtlasDatabase",
	ClassID_AudioBuildInfo:                         "AudioBuildInfo",
	ClassID_CachedSpriteAtlasRuntimeData:           "CachedSpriteAtlasRuntimeData",
	ClassID_RendererFake:                           "RendererFake",
	ClassID_AssemblyDefinitionReferenceAsset:       "AssemblyDefinitionReferenceAsset",
	ClassID_BuiltAssetBundleInfoSet:                "BuiltAssetBundleInfoSet",
	ClassID_SpriteAtlas:                            "SpriteAtlas",
	ClassID_RayTracingShaderImporter:               "RayTracingShaderImporter",
	ClassID_PreviewImporter:                        "PreviewImporter",
	ClassID_RayTracingShader:                       "RayTracingShader",
	ClassID_LightingSettings:                       "LightingSettings",
	ClassID_PlatformModuleSetup:                    "PlatformModuleSetup",
	ClassID_VersionControlSettings:                 "VersionControlSettings",
	ClassID_AimConstraint:                          "AimConstraint",
	ClassID_VFXManager:                             "VFXManager",
	ClassID_VisualEffectSubgraph:                   "VisualEffectSubgraph",
	ClassID_RuleSetFileAsset:                       "RuleSetFileAsset",
	ClassID_VisualEffectSubgraphOperator:           "VisualEffectSubgraphOperator",
	ClassID_VisualEffectSubgraphBlock:              "VisualEffectSubgraphBlock",
	ClassID_Prefab:                                 "Prefab",
	ClassID_LocalizationImporter:                   "LocalizationImporter",
	ClassID_Derived:                                "Derived",
	ClassID_PropertyModificationsTargetTestObject:  "PropertyModificationsTargetTestObject",
	ClassID_ReferencesArtifactGenerator:            "ReferencesArtifactGenerator",
	ClassID_AssemblyDefinitionAsset:                "AssemblyDefinitionAsset",
	ClassID_SceneVisibilityState:                   "SceneVisibilityState",
	ClassID_LookAtConstraint:                       "LookAtConstraint",
	ClassID_SpriteAtlasImporter:                    "SpriteAtlasImporter",
	ClassID_MultiArtifactTestImporter:              "MultiArtifactTestImporter",
	ClassID_GameObjectRecorder:                     "GameObjectRecorder",
	ClassID_LightingDataAssetParent:                "LightingDataAssetParent",
	ClassID_PresetManager:                          "PresetManager",
	ClassID_TestObjectWithSpecialLayoutTwo:         "TestObjectWithSpecialLayoutTwo",
	ClassID_StreamingManager:                       "StreamingManager",
	ClassID_LowerResBlitTexture:                    "LowerResBlitTexture",
	ClassID_StreamingController:                    "StreamingController",
	ClassID_RenderPassAttachment:                   "RenderPassAttachment",
	ClassID_TestObjectVectorPairStringBool:         "TestObjectVectorPairStringBool",
	ClassID_GridLayout:                             "GridLayout",
	ClassID_AssemblyDefinitionImporter:             "AssemblyDefinitionImporter",
	ClassID_ParentConstraint:                       "ParentConstraint",
	ClassID_FakeComponent:                          "FakeComponent",
	ClassID_RuleSetFileImporter:                    "RuleSetFileImporter",
	ClassID_PositionConstraint:                     "PositionConstraint",
	ClassID_RotationConstraint:                     "RotationConstraint",
	ClassID_ScaleConstraint:                        "ScaleConstraint",
	ClassID_Tilemap:                                "Tilemap",
	ClassID_PackageManifest:                        "PackageManifest",
	ClassID_PackageManifestImporter:                "PackageManifestImporter",
	ClassID_TerrainLayer:                           "TerrainLayer",
	ClassID_SpriteShapeRenderer:                    "SpriteShapeRenderer",
	ClassID_NativeObjectType:                       "NativeObjectType",
	ClassID_TestObjectWithSerializedMapStringBool:  "TestObjectWithSerializedMapStringBool",
	ClassID_SerializableManagedHost:                "SerializableManagedHost",
	ClassID_VisualEffectAsset:                      "VisualEffectAsset",
	ClassID_VisualEffectImporter:                   "VisualEffectImporter",
	ClassID_VisualEffectResource:                   "VisualEffectResource",
	ClassID_VisualEffectObject:                     "VisualEffectObject",
	ClassID_VisualEffect:                           "VisualEffect",
	ClassID_LocalizationAsset:                      "LocalizationAsset",
	ClassID_ScriptedImporter:                       "ScriptedImporter",
	ClassID_TilemapEditorUserSettings:              "TilemapEditorUserSettings",
}

func (id AssetsClassID) String() string {
	if name, ok := AssetsClassIDName[id]; ok {
		return name
	}
	return fmt.Sprintf("AssetsClassID(%d)", int(id))
}
